package vault

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	hashivault "github.com/hashicorp/vault/api"
)

const (
	loggingKeySecretName    = "secret"
	loggingKeyResult        = "result"
	loggingKeyRenewedAt     = "renewed_at"
	loggingKeyLeaseDuration = "lease_duration"
)

// renewResult is a bitmask which could contain one or more of the values below
type renewResult uint8

const (
	renewError renewResult = 1 << iota
	exitRequested
	expiring // will be revoked soon
)

// RenewLease Once you've set the token for your Vault client, you will need to
// periodically renew it. Likewise, the database credentials lease will expire
// at some point and also needs to be renewed periodically.
//
// A function like this one should be run as a goroutine to avoid blocking.
// Production applications may also need to be more tolerant of failures and
// retry on errors rather than exiting.
//
// Additionally, enterprise Vault users should be aware that due to eventual
// consistency, the API may return unexpected errors when running Vault with
// performance standbys or performance replication, despite the client having
// a freshly renewed token. See the link below for several ways to mitigate
// this which are outside the scope of this code sample.
//
// ref: https://www.vaultproject.io/docs/enterprise/consistency#vault-1-7-mitigations
func RenewLease(
	ctx context.Context,
	l *slog.Logger,
	client ClientGetter,
	name string,
	credentials *hashivault.Secret,
	renewFunc RenewalFunc,
) error {
	l.Debug("renewing lease", slog.String(loggingKeySecretName, name))

	currentCredentials := credentials

	for {
		res, err := renewLeaseOnce(ctx, l, client, name, currentCredentials)
		if err != nil {
			return fmt.Errorf("unable to renew lease: %w", err)
		} else if res&exitRequested != 0 {
			// Context was cancelled. Program is exiting.
			l.Debug("exit requested", slog.String(loggingKeySecretName, name))
			return nil
		}

		if err := handleWatcherResult(l, res, func() error {
			newCredentials, err := renewFunc()
			if err != nil {
				return fmt.Errorf("unable to renew credentials: %w", err)
			}
			currentCredentials = newCredentials
			return nil
		}); err != nil {
			return fmt.Errorf("unable to handle watcher result: %w", err)
		}

		l.Debug("lease renewed", slog.String(loggingKeySecretName, name))
	}
}

// leaseRenew attempts to renew the given credentials once, returning the result of the attempt.
func renewLeaseOnce(
	ctx context.Context,
	l *slog.Logger,
	client ClientGetter,
	name string,
	credentials *hashivault.Secret,
) (renewResult, error) {
	credentialsWatcher, err := client.Client().NewLifetimeWatcher(&hashivault.LifetimeWatcherInput{
		Secret:    credentials,
		Increment: int(time.Hour.Seconds()),
	})
	if err != nil {
		return renewError, fmt.Errorf("unable to initialize credentials lifetime watcher: %w", err)
	}

	go credentialsWatcher.Start()
	defer credentialsWatcher.Stop()

	res, err := monitorWatcher(ctx, l, name, credentialsWatcher)
	if err != nil {
		return renewError, fmt.Errorf("unable to monitor watcher: %w", err)
	}

	return res, nil
}

// monitorWatcher monitors the renewal of a Vault token and returns a channel
func monitorWatcher(ctx context.Context, l *slog.Logger, token string, watcher *hashivault.LifetimeWatcher) (renewResult, error) {
	for {
		select {
		case <-ctx.Done():
			return exitRequested, nil

			// DoneCh will return if renewal fails, or if the remaining lease
			// duration is under a built-in threshold and either renewing is not
			// extending it or renewing is disabled.  In both cases, the caller
			// should attempt a re-read of the secret. Clients should check the
			// return value of the channel to see if renewal was successful.
		case err := <-watcher.DoneCh():
			// Leases created by a token get revoked when the token is revoked.
			return expiring, err

			// RenewCh is a channel that receives a message when a successful
			// renewal takes place and includes metadata about the renewal.
		case info := <-watcher.RenewCh():
			l.Debug("renewal successful",
				slog.String(loggingKeyRenewedAt, info.RenewedAt.String()),
				slog.String(loggingKeySecretName, token),
				slog.String(loggingKeyLeaseDuration, fmt.Sprintf("%ds", info.Secret.LeaseDuration)),
			)
		}
	}
}

// handleWatcherResult processes the result of the watcher and executes
func handleWatcherResult(l *slog.Logger, result renewResult, onExpire ...func() error) error {
	switch {
	case result&exitRequested != 0:
		l.Debug("result is exitRequested", slog.Int(loggingKeyResult, int(result)))
		return nil
	case result&expiring != 0:
		l.Debug("result is expiring", slog.Int(loggingKeyResult, int(result)))
		for _, f := range onExpire {
			if err := f(); err != nil {
				return fmt.Errorf("onExpire function failed: %w", err)
			}
		}
		return nil
	default:
		l.Debug("no action required", slog.Int(loggingKeyResult, int(result)))
		return nil
	}
}
