package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/gorilla/mux"
	"github.com/jacobbrewer1/goredis"
	"github.com/jacobbrewer1/uhttp"
	"github.com/jacobbrewer1/vaulty"
	"github.com/jacobbrewer1/vaulty/repositories"
	"github.com/jacobbrewer1/web/cache"
	"github.com/jacobbrewer1/web/logging"
	"github.com/jacobbrewer1/web/utils"
	"github.com/jacobbrewer1/workerpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	kubeCache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
)

const (
	MetricsPort = 9090
	HealthPort  = 9091
)

var (
	// ErrNilLogger is returned when a nil logger is passed to NewApp.
	ErrNilLogger = errors.New("logger is nil")
)

type (
	AppConfig struct {
		ConfigLocation string `env:"CONFIG_LOCATION" envDefault:"config.json"`
	}

	App struct {
		// l is the logger for the application.
		l *slog.Logger

		// baseCfg is the base configuration for the application.
		baseCfg *AppConfig

		// vaultClient is the vault client for the application.
		vaultClient vaulty.Client

		// baseCtx is the base context for the application.
		baseCtx context.Context

		// baseCtxCancel is the base context cancel function.
		baseCtxCancel context.CancelFunc

		// vip is the viper instance for the application.
		vip *viper.Viper

		// metricsEnabled is a flag to enable metrics for the application.
		metricsEnabled bool

		// servers is the list of servers for the application.
		servers sync.Map

		// shutdownWg is used to wait for all shutdown tasks to complete.
		shutdownWg *sync.WaitGroup

		// db is the database for the application.
		db *repositories.Database

		// kubeClient interacts with the Kubernetes API server.
		kubeClient kubernetes.Interface

		// kubernetesInformerFactory is a factory used for initialising a pod informer.
		kubernetesInformerFactory informers.SharedInformerFactory

		// podInformer is an informer for Kubernetes Pod objects.
		podInformer kubeCache.SharedIndexInformer

		// podLister is a lister for Kubernetes Pod objects.
		podLister listersv1.PodLister

		// leaderElection is the leader election for the application.
		leaderElection *leaderelection.LeaderElector

		// leaderChange is the channel that is notified when the leader changes.
		leaderChange chan struct{}

		// redisPool is the redis pool for the application.
		redisPool goredis.Pool

		// workerPool is the worker pool that can execute tasks concurrently.
		workerPool workerpool.Pool

		// indefiniteAsyncTasks is the list of indefinite async tasks for the application.
		indefiniteAsyncTasks sync.Map

		// serviceEndpointHashBucket is the service endpoint hash bucket for the application.
		serviceEndpointHashBucket *cache.ServiceEndpointHashBucket

		// natsClient is the nats client for the application.
		natsClient *nats.Conn

		// natsJetStream is the nats jetstream for the application.
		natsJetStream jetstream.JetStream

		// natStream is the nats stream for the application.
		natsStream jetstream.Stream
	}
)

// NewApp creates a new application with the given logger.
func NewApp(l *slog.Logger) (*App, error) {
	if l == nil {
		return nil, ErrNilLogger
	}

	baseCtx, baseCtxCancel := utils.GetInterruptedContext(l)

	baseCfg := new(AppConfig)
	if err := env.Parse(baseCfg); err != nil {
		return nil, fmt.Errorf("failed to parse app config: %w", err)
	}

	vip := viper.New()
	vip.SetConfigFile(baseCfg.ConfigLocation)
	if err := vip.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	return &App{
		l:              l,
		baseCfg:        baseCfg,
		baseCtx:        baseCtx,
		baseCtxCancel:  baseCtxCancel,
		vip:            vip,
		metricsEnabled: true,
		shutdownWg:     new(sync.WaitGroup),
	}, nil
}

// Start starts the application and applies the given options.
func (a *App) Start(opts ...StartOption) error {
	a.l.Info("starting application",
		slog.String(logging.KeyGitCommit, utils.GitCommit()),
		slog.String(logging.KeyRuntime, fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)),
		slog.String(logging.KeyBuildDate, utils.CommitTimestamp().String()),
	)

	for _, opt := range opts {
		if err := opt(a); err != nil {
			return fmt.Errorf("failed to apply option: %w", err)
		}
	}

	if a.metricsEnabled {
		metricsRouter := mux.NewRouter()
		metricsRouter.Handle("/metrics", promhttp.Handler())
		a.servers.Store("metrics", &http.Server{
			Addr:              fmt.Sprintf(":%d", MetricsPort),
			Handler:           metricsRouter,
			ReadHeaderTimeout: 10 * time.Second,
		})
	}

	if a.leaderElection != nil {
		go a.leaderElection.Run(a.baseCtx)
	}

	a.servers.Range(func(name, srv any) bool {
		serverName, ok := name.(string)
		if !ok {
			a.l.Error("failed to cast server name to string")
			return false
		}

		server, ok := srv.(*http.Server)
		if !ok {
			a.l.Error("failed to cast server to http.Server")
			return false
		}

		a.startServer(serverName, server)
		return true
	})

	a.indefiniteAsyncTasks.Range(func(name, fn any) bool {
		taskName, ok := name.(string)
		if !ok {
			a.l.Error("failed to cast task name to string")
			return false
		}

		taskFn, ok := fn.(AsyncTaskFunc)
		if !ok {
			a.l.Error("failed to cast task function to AsyncTaskFunc")
			return false
		}

		a.startAsyncTask(taskName, true, taskFn)
		return true
	})

	return nil
}

// startServer starts the given server.
func (a *App) startServer(name string, srv *http.Server) {
	l := a.l.With(slog.String(logging.KeyServer, name))

	a.shutdownWg.Add(1)
	go func() {
		defer a.shutdownWg.Done()

		l.Info("server listening")
		if err := srv.ListenAndServe(); errors.Is(err, http.ErrServerClosed) {
			l.Info("server shut down gracefully", slog.Any(logging.KeyError, err))
		} else {
			l.Error("server closed", slog.Any(logging.KeyError, err))
		}
	}()
}

// ChildContext returns a child context of the base context.
func (a *App) ChildContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(a.baseCtx)
}

// WaitForEnd waits for the application to complete, either normally or via an interrupt signal.
func (a *App) WaitForEnd(onEnd ...func()) {
	<-a.baseCtx.Done()

	for _, fn := range onEnd {
		fn()
	}
}

// Shutdown stops the application.
func (a *App) Shutdown() {
	if a.baseCtxCancel != nil {
		a.baseCtxCancel()
	}

	ctx, cancel := context.WithTimeout(a.baseCtx, 15*time.Second)
	defer cancel()

	a.servers.Range(func(name, srv any) bool {
		server, ok := srv.(*http.Server)
		if !ok {
			a.l.Error("failed to cast server to http.Server")
			return false
		}

		nameStr, ok := name.(string)
		if !ok {
			a.l.Warn("failed to cast server name to string")
			nameStr = "unknown"
		}

		if err := server.Shutdown(ctx); err != nil {
			a.l.Error("failed to shutdown server",
				slog.String(logging.KeyServer, nameStr),
				slog.Any(logging.KeyError, err),
			)
		}

		return true
	})

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.l.Error("failed to close database", slog.Any(logging.KeyError, err))
		}
	}

	if a.redisPool != nil {
		if err := a.redisPool.Conn().Close(); err != nil {
			a.l.Error("failed to close redis pool", slog.Any(logging.KeyError, err))
		}
	}

	if a.workerPool != nil {
		a.workerPool.Stop()
	}

	a.shutdownWg.Wait()
}

// Logger returns the logger for the application.
func (a *App) Logger() *slog.Logger {
	return a.l
}

// VaultClient returns the vault client for the application.
func (a *App) VaultClient() vaulty.Client {
	return a.vaultClient
}

// Viper returns the viper instance for the application.
func (a *App) Viper() *viper.Viper {
	return a.vip
}

// DBConn returns the database connection for the application.
func (a *App) DBConn() *repositories.Database {
	return a.db
}

// KubeClient returns the Kubernetes client for the application.
func (a *App) KubeClient() kubernetes.Interface {
	return a.kubeClient
}

// Done returns a channel that is closed when the application is done.
func (a *App) Done() <-chan struct{} {
	return a.baseCtx.Done()
}

// IsLeader returns true if the application is the leader.
func (a *App) IsLeader() bool {
	if a.leaderElection == nil {
		a.l.Info("leader election not set, assuming leader")
		return true
	}

	return a.leaderElection.IsLeader()
}

// LeaderChange returns a channel that is notified when the leader changes.
func (a *App) LeaderChange() <-chan struct{} {
	return a.leaderChange
}

// StartServer starts a new server with the given name and http.Server.
func (a *App) StartServer(name string, srv *http.Server) error {
	if _, found := a.servers.Load(name); found {
		return fmt.Errorf("server %s already exists", name)
	}

	// If the server handler is gorilla mux, check the not found handler and method not allowed handler
	if muxRouter, ok := srv.Handler.(*mux.Router); ok {
		if muxRouter.NotFoundHandler == nil {
			a.l.Info("not found handler not set for server, applying default handler")
			muxRouter.NotFoundHandler = uhttp.NotFoundHandler()
		}
		if muxRouter.MethodNotAllowedHandler == nil {
			a.l.Info("method not allowed handler not set for server, applying default handler")
			muxRouter.MethodNotAllowedHandler = uhttp.MethodNotAllowedHandler()
		}
	}

	a.servers.Store(name, srv)
	a.startServer(name, srv)
	return nil
}

// startAsyncTask starts async task f.
func (a *App) startAsyncTask(name string, indefinite bool, fn AsyncTaskFunc) {
	a.l.Info("starting async task", slog.String(logging.KeyName, name))
	a.shutdownWg.Add(1)
	go func() {
		defer a.shutdownWg.Done()
		fn(a.baseCtx)

		// If task is configured as indefinite and the task stops before we stop running the entire app, close the app down
		// with an error.
		if indefinite && !errors.Is(a.baseCtx.Err(), context.Canceled) { // nolint:revive // Traditional error handling
			a.l.Warn("indefinite async task closed before app shutdown",
				slog.String(logging.KeyName, name),
			)
			a.baseCtxCancel()
		}
	}()
}

// RedisPool returns the redis pool for the application.
func (a *App) RedisPool() goredis.Pool {
	return a.redisPool
}

// WorkerPool returns the worker pool for the application.
func (a *App) WorkerPool() workerpool.Pool {
	return a.workerPool
}

// NatsClient returns the NATS client for the application.
func (a *App) NatsClient() *nats.Conn {
	return a.natsClient
}

// NatsJetStream returns the JetStream stream for the application.
func (a *App) NatsJetStream() jetstream.JetStream {
	return a.natsJetStream
}

// NatsStream returns the NATS stream for the application.
func (a *App) NatsStream() jetstream.Stream {
	return a.natsStream
}

// CreateNatsJetStreamConsumer creates a new JetStream consumer with the given name and subject filter.
func (a *App) CreateNatsJetStreamConsumer(consumerName, subjectFilter string) (jetstream.Consumer, error) {
	if a.natsStream == nil {
		return nil, errors.New("nats stream is nil, ensure WithNatsJetStream is called")
	}

	cons, err := a.natsStream.CreateOrUpdateConsumer(a.baseCtx, jetstream.ConsumerConfig{
		Durable:       consumerName,                // Durable name ensures state retention
		AckPolicy:     jetstream.AckExplicitPolicy, // Acknowledge messages explicitly
		FilterSubject: subjectFilter,               // Only accept messages from this subject
		DeliverPolicy: jetstream.DeliverAllPolicy,  // Deliver all messages
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer: %w", err)
	}

	return cons, nil
}

// ServiceEndpointHashBucket returns the service endpoint hash bucket for the application.
func (a *App) ServiceEndpointHashBucket() *cache.ServiceEndpointHashBucket {
	return a.serviceEndpointHashBucket
}

// PodLister returns the pod lister for the application.
func (a *App) PodLister() listersv1.PodLister {
	return a.podLister
}

// PodInformer returns the pod informer for the application.
func (a *App) PodInformer() kubeCache.SharedIndexInformer {
	return a.podInformer
}

// KubernetesInformerFactory returns the Kubernetes informer factory for the application.
func (a *App) KubernetesInformerFactory() informers.SharedInformerFactory {
	return a.kubernetesInformerFactory
}
