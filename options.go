package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"

	"github.com/jacobbrewer1/goredis"
	"github.com/jacobbrewer1/vaulty"
	"github.com/jacobbrewer1/vaulty/vsql"
	"github.com/jacobbrewer1/web/cache"
	"github.com/jacobbrewer1/web/health"
	"github.com/jacobbrewer1/web/k8s"
	"github.com/jacobbrewer1/web/logging"
	"github.com/jacobbrewer1/workerpool"
)

const (
	// inClusterNatsEndpoint is the default NATS endpoint for in-cluster communication.
	inClusterNatsEndpoint = "nats://nats-headless.nats:4222"

	// leaderElectionLeaseDuration specifies the duration that non-leader candidates
	// will wait to forcefully acquire leadership if the current leader fails to renew.
	leaderElectionLeaseDuration = 15 * time.Second

	// leaderElectionRenewDeadline specifies the duration that the acting leader
	// will attempt to renew its leadership before giving up.
	leaderElectionRenewDeadline = 10 * time.Second

	// leaderElectionRetryPeriod specifies the interval between retries for leader election actions.
	leaderElectionRetryPeriod = 2 * time.Second
)

var (
	// ErrNoHostname is a predefined error that indicates the hostname is not set.
	ErrNoHostname = errors.New("no hostname provided")
)

// AsyncTaskFunc defines a function type for asynchronous tasks.
type AsyncTaskFunc = func(context.Context)

// StartOption defines a function type for configuring the application during startup.
type StartOption = func(*App) error

// WithViperConfig is a StartOption that sets up the viper configuration.
func WithViperConfig() StartOption {
	return func(a *App) error {
		vip := viper.New()
		vip.SetConfigFile(a.baseCfg.ConfigLocation)
		if err := vip.ReadInConfig(); err != nil {
			return fmt.Errorf("error reading config file into viper: %w", err)
		}
		a.vip = vip
		return nil
	}
}

// WithConfigWatchers is a StartOption that registers functions to be called when the config file changes.
func WithConfigWatchers(fn ...func()) StartOption {
	return func(a *App) error {
		vip := a.Viper()
		vip.OnConfigChange(func(e fsnotify.Event) {
			a.l.Info("Config file changed", slog.String(logging.KeyFile, e.Name))
			for _, f := range fn {
				f()
			}
		})
		vip.WatchConfig()
		return nil
	}
}

// WithVaultClient is a StartOption that sets up the vault client.
func WithVaultClient() StartOption {
	return func(a *App) error {
		vip := a.Viper()
		vc, err := VaultClient(a.baseCtx, logging.LoggerWithComponent(a.l, "vault"), vip)
		if err != nil {
			return fmt.Errorf("error getting vault client: %w", err)
		}

		a.vaultClient = vc
		return nil
	}
}

// WithDatabaseFromVault is a StartOption that sets up the database connection using Vault secrets.
func WithDatabaseFromVault() StartOption {
	return func(a *App) error {
		vc := a.VaultClient()
		vip := a.Viper()

		vs, err := vc.Path(
			vip.GetString("vault.database.role"),
			vaulty.WithPrefix(vip.GetString("vault.database.path")),
		).GetSecret(a.baseCtx)
		if errors.Is(err, vaulty.ErrSecretNotFound) {
			return fmt.Errorf("secrets not found in vault: %s", vip.GetString("vault.database.path"))
		} else if err != nil {
			return fmt.Errorf("error getting secrets from vault: %w", err)
		}

		a.db, err = vsql.ConnectDB(
			a.baseCtx,
			logging.LoggerWithComponent(a.l, "database_connector"),
			vc,
			vip,
			vs,
		)
		if err != nil {
			return fmt.Errorf("error creating database connector: %w", err)
		}
		return nil
	}
}

// WithInClusterKubeClient is a StartOption that sets up the in-cluster Kubernetes client.
func WithInClusterKubeClient() StartOption {
	return func(a *App) error {
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("failed to get in-cluster config: %w", err)
		}

		kubeClient, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to create kube client: %w", err)
		}

		a.kubeClient = kubeClient
		return nil
	}
}

// WithLeaderElection is a StartOption that sets up leader election using Kubernetes lease locks.
//
// This function configures leader election for the application using Kubernetes' lease lock mechanism.
// It ensures that only one instance of the application acts as the leader at any given time.
func WithLeaderElection(lockName string) StartOption {
	return func(a *App) error {
		switch {
		case k8s.PodName() == "":
			return ErrNoHostname
		case lockName == "":
			return errors.New("lock name cannot be empty")
		}

		kubeClient := a.KubeClient()

		klog.SetSlogLogger(logging.LoggerWithComponent(a.l, "klog"))

		a.leaderChange = make(chan struct{})

		// Create the leader election
		le, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
			Lock: &resourcelock.LeaseLock{
				LeaseMeta: v1.ObjectMeta{
					Name:      lockName,
					Namespace: k8s.DeployedNamespace(),
				},
				Client: kubeClient.CoordinationV1(),
				LockConfig: resourcelock.ResourceLockConfig{
					Identity: k8s.PodName(),
				},
			},
			LeaseDuration: leaderElectionLeaseDuration,
			RenewDeadline: leaderElectionRenewDeadline,
			RetryPeriod:   leaderElectionRetryPeriod,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					a.l.Info("Started leading")
				},
				OnStoppedLeading: func() {
					a.l.Info("Stopped leading")
				},
				OnNewLeader: func(identity string) {
					a.l.Info("Leader Changed",
						slog.String(logging.KeyIdentity, identity),
					)

					select {
					case a.leaderChange <- struct{}{}:
					default:
						// Prevent blocking
					}
				},
			},
			ReleaseOnCancel: true,
		})
		if err != nil {
			return fmt.Errorf("failed to create leader election: %w", err)
		}

		a.leaderElection = le
		return nil
	}
}

// WithHealthCheck is a StartOption that sets up the health check server.
func WithHealthCheck(checks ...*health.Check) StartOption {
	return func(a *App) error {
		if _, exists := a.servers.Load("health"); exists {
			return errors.New("health check server already registered")
		}

		// No grace period as we do not want to receive any traffic the moment the pod health check fails.
		readinessChecker, err := health.NewChecker()
		if err != nil {
			return fmt.Errorf("error creating health checker: %w", err)
		}

		// Setting a grace period for liveness checks as Kubernetes will kill the pod
		// if the liveness check fails. This allows for a grace period before the pod is killed.
		livenessChecker, err := health.NewChecker(health.WithCheckerErrorGracePeriod(10 * time.Second))
		if err != nil {
			return fmt.Errorf("error creating liveness checker: %w", err)
		}

		for _, check := range checks {
			if err := readinessChecker.AddCheck(check); err != nil {
				return fmt.Errorf("error adding health check %s: %w", check.String(), err)
			}

			if err := livenessChecker.AddCheck(check); err != nil {
				return fmt.Errorf("error adding liveness check %s: %w", check.String(), err)
			}
		}

		r := mux.NewRouter()
		r.Handle("/readyz", readinessChecker.Handler()).Methods(http.MethodGet)
		r.Handle("/livez", livenessChecker.Handler()).Methods(http.MethodGet)
		a.servers.Store("health", &http.Server{
			Addr:              fmt.Sprintf(":%d", HealthPort),
			Handler:           r,
			ReadHeaderTimeout: httpReadHeaderTimeout,
		})
		return nil
	}
}

// WithRedisPool is a StartOption that sets up the Redis connection pool.
func WithRedisPool() StartOption {
	return func(a *App) error {
		vip := a.Viper()
		vc := a.VaultClient()

		keydbPath := vip.GetString("vault.keydb.name")
		keydbSecret, err := vc.Path(keydbPath).GetKvSecretV2(a.baseCtx)
		if errors.Is(err, vaulty.ErrSecretNotFound) {
			return fmt.Errorf("keydb secrets not found in vault path: %s", keydbPath)
		} else if err != nil {
			return fmt.Errorf("error getting keydb secrets from vault: %w", err)
		}

		redisPassword, ok := keydbSecret.Data["password"].(string)
		if !ok {
			return fmt.Errorf("invalid password type in keydb secret at path: %s", keydbPath)
		}

		rp, err := goredis.NewPool(
			goredis.WithLogger(logging.LoggerWithComponent(a.l, "goredis")),
			goredis.WithAddress(vip.GetString("keydb.address")),
			goredis.WithNetwork(vip.GetString("keydb.network")),
			goredis.WithDialOpts(
				redis.DialPassword(redisPassword),
				redis.DialDatabase(vip.GetInt("keydb.database")),
			),
		)
		if err != nil {
			return fmt.Errorf("error creating redis pool: %w", err)
		}

		a.redisPool = rp
		return nil
	}
}

// WithMetricsEnabled is a StartOption that enables or disables metrics for the application.
func WithMetricsEnabled(metricsEnabled bool) StartOption {
	return func(a *App) error {
		a.metricsEnabled = metricsEnabled
		return nil
	}
}

// WithWorkerPool is a StartOption that sets up the worker pool for the application.
func WithWorkerPool() StartOption {
	return func(a *App) error {
		wp := workerpool.New(
			workerpool.WithDelayedStart(),
		)

		a.workerPool = wp
		return nil
	}
}

// WithDependencyBootstrap is a StartOption that bootstraps application dependencies.
//
// This function allows the custom dependency bootstrapping, which is executed during the application startup process.
func WithDependencyBootstrap(fn func(ctx context.Context) error) StartOption {
	return func(a *App) error {
		return fn(a.baseCtx)
	}
}

// WithIndefiniteAsyncTask is a StartOption that sets up an indefinite asynchronous task.
func WithIndefiniteAsyncTask(name string, fn AsyncTaskFunc) StartOption {
	return func(a *App) error {
		a.indefiniteAsyncTasks.Store(name, fn)
		return nil
	}
}

// WithFixedHashBucket is a StartOption that sets up a fixed-size hash bucket.
func WithFixedHashBucket(size uint) StartOption {
	return func(a *App) error {
		hb := cache.NewFixedHashBucket(size)
		a.fixedHashBucket = hb
		return nil
	}
}

// WithServiceEndpointHashBucket is a StartOption that sets up the service endpoint hash bucket.
func WithServiceEndpointHashBucket(appName string) StartOption {
	return func(a *App) error {
		sb := cache.NewServiceEndpointHashBucket(
			logging.LoggerWithComponent(a.l, "service_endpoint_hash_bucket"),
			a.KubeClient(),
			appName,
			k8s.DeployedNamespace(),
			k8s.PodName(),
		)

		a.serviceEndpointHashBucket = sb

		return sb.Start(a.baseCtx)
	}
}

// WithNatsClient is a StartOption that sets up the NATS client.
func WithNatsClient(target string) StartOption {
	return func(a *App) error {
		if target == "" {
			return errors.New("target cannot be empty")
		}

		nc, err := nats.Connect(target)
		if err != nil {
			return fmt.Errorf("failed to connect to nats: %w", err)
		}

		a.natsClient = nc
		return nil
	}
}

// WithInClusterNatsClient is a StartOption that sets up the NATS client with the in-cluster endpoint.
//
// This function is a wrapper around `WithNatsClient` that uses the default in-cluster NATS endpoint.
func WithInClusterNatsClient() StartOption {
	return WithNatsClient(inClusterNatsEndpoint)
}

// WithNatsJetStream is a StartOption that sets up NATS JetStream with the given stream name, retention policy, and subjects.
func WithNatsJetStream(streamName string, retentionPolicy jetstream.RetentionPolicy, subjects []string) StartOption {
	return func(a *App) error {
		natsClient := a.NatsClient()

		js, err := jetstream.New(natsClient)
		if err != nil {
			return fmt.Errorf("failed to create jetstream: %w", err)
		}
		a.natsJetStream = js

		_, err = js.CreateOrUpdateStream(a.baseCtx, jetstream.StreamConfig{
			Name:      streamName,
			Subjects:  subjects,
			Storage:   jetstream.FileStorage,
			Retention: retentionPolicy,
		})
		if err != nil && !errors.Is(err, jetstream.ErrStreamNameAlreadyInUse) {
			return fmt.Errorf("failed to create stream: %w", err)
		}

		a.natsStream, err = js.Stream(a.baseCtx, streamName)
		if err != nil {
			return fmt.Errorf("failed to get stream: %w", err)
		}

		return nil
	}
}

// WithKubernetesPodInformer is a StartOption that sets up a Kubernetes Pod informer.
//
// This function initializes a Kubernetes SharedInformerFactory and creates an informer
// for Kubernetes Pod objects. The informer is used to watch and cache Pod resources
// in the Kubernetes cluster.
func WithKubernetesPodInformer(informerOptions ...informers.SharedInformerOption) StartOption {
	return func(a *App) error {
		initKubernetesInformerFactory(a, informerOptions...)

		a.l.Info("creating kubernetes pod informer")
		base := a.kubernetesInformerFactory.Core().V1().Pods()
		a.podInformer = base.Informer()
		a.podLister = base.Lister()
		return nil
	}
}

// WithKubernetesSecretInformer is a StartOption that sets up a Kubernetes Secret informer.
//
// This function initializes a Kubernetes SharedInformerFactory and creates an informer
// for Kubernetes Secret objects. The informer is used to watch and cache Secret resources
// in the Kubernetes cluster.
func WithKubernetesSecretInformer(informerOptions ...informers.SharedInformerOption) StartOption {
	return func(a *App) error {
		initKubernetesInformerFactory(a, informerOptions...)

		a.l.Info("creating kubernetes secret informer")
		base := a.kubernetesInformerFactory.Core().V1().Secrets()
		a.secretInformer = base.Informer()
		a.secretLister = base.Lister()
		return nil
	}
}

// WithKubernetesConfigMapInformer is a StartOption that sets up a Kubernetes ConfigMap informer.
//
// This function initializes a Kubernetes SharedInformerFactory and creates an informer
// for Kubernetes ConfigMap objects. The informer is used to watch and cache ConfigMap resources
// in the Kubernetes cluster.
func WithKubernetesConfigMapInformer(informerOptions ...informers.SharedInformerOption) StartOption {
	return func(a *App) error {
		initKubernetesInformerFactory(a, informerOptions...)

		a.l.Info("creating kubernetes configmap informer")
		base := a.kubernetesInformerFactory.Core().V1().ConfigMaps()
		a.configMapInformer = base.Informer()
		a.configMapLister = base.Lister()
		return nil
	}
}
