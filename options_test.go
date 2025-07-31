package web

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestWithViperConfig(t *testing.T) {
	t.Run("golden", func(t *testing.T) {
		file, err := os.CreateTemp(t.TempDir(), "viper-config*.json")
		require.NoError(t, err, "expected no error when creating temp file for viper config")

		t.Setenv("CONFIG_LOCATION", file.Name())

		_, err = file.WriteString(`{
  "vault": {
    "address": "https://vault.domain.com",
    "token": "root",
    "database": {
      "path": "mount/creds",
      "role": "role"
    },
    "transit": {
      "name": "transit-mount",
      "key": "key-name"
    },
    "kvv2_mount": "kvv2-mount",
    "keydb": {
      "name": "keydb"
    }
  },
  "database": {
    "host": "192.168.0.1:3306",
    "schema": "test_schema"
  },
  "keydb": {
    "max_idle": 10,
    "max_active": 20,
    "idle_timeout_secs": 300,
    "address": "localhost:6379",
    "db": 0,
    "network": "tcp"
  }
}`)

		a := newTestApp(t)
		err = a.Start(WithViperConfig())
		require.NoError(t, err, "expected no error when starting app with viper config")

		var v *viper.Viper
		require.NotPanics(t, func() {
			v = a.Viper()
		})
		require.NotNil(t, v, "expected viper instance to be initialized")

		require.Equal(t, "https://vault.domain.com", v.GetString("vault.address"), "expected vault address to match")
		require.Equal(t, "root", v.GetString("vault.token"), "expected vault token to match")
		require.Equal(t, "mount/creds", v.GetString("vault.database.path"), "expected vault database path to match")
		require.Equal(t, "role", v.GetString("vault.database.role"), "expected vault database role to match")
		require.Equal(t, "transit-mount", v.GetString("vault.transit.name"), "expected vault transit mount name to match")
		require.Equal(t, "key-name", v.GetString("vault.transit.key"), "expected vault transit key to match")
		require.Equal(t, "kvv2-mount", v.GetString("vault.kvv2_mount"), "expected vault kvv2 mount to match")
		require.Equal(t, "keydb", v.GetString("vault.keydb.name"), "expected vault keydb name to match")
	})

	t.Run("no_config_file", func(t *testing.T) {
		a := newTestApp(t)
		err := a.Start(WithViperConfig())
		require.EqualError(t, err, "failed to apply option: error reading config file into viper: open config.json: no such file or directory", "expected specific error when no config file is found")
	})
}

func TestWithConfigWatchers(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "viper-config*.json")
	require.NoError(t, err, "expected no error when creating temp file for viper config")

	_, err = file.WriteString(`{
  "vault": {
    "address": "https://vault.domain.com",
    "token": "root",
    "database": {
      "path": "mount/creds",
      "role": "role"
    },
    "transit": {
      "name": "transit-mount",
      "key": "key-name"
    },
    "kvv2_mount": "kvv2-mount",
    "keydb": {
      "name": "keydb"
    }
  },
  "database": {
    "host": "192.168.0.1:3306",
    "schema": "test_schema"
  },
  "keydb": {
    "max_idle": 10,
    "max_active": 20,
    "idle_timeout_secs": 300,
    "address": "localhost:6379",
    "db": 0,
    "network": "tcp"
  }
}`)

	t.Setenv("CONFIG_LOCATION", file.Name())

	a := newTestApp(t)

	var configWatcherCalled int32
	err = a.Start(WithViperConfig(), WithConfigWatchers(func(ctx context.Context) {
		atomic.StoreInt32(&configWatcherCalled, 1)
	}))
	require.NoError(t, err, "expected no error when starting app with viper config and watchers")
	require.Equal(t, int32(0), atomic.LoadInt32(&configWatcherCalled), "expected config watcher not to be called immediately")

	// Simulate a config change by writing to the file
	err = file.Truncate(0)
	require.NoError(t, err, "expected no error when truncating config file")
	_, err = file.Seek(0, 0)
	require.NoError(t, err, "expected no error when seeking to the beginning of config file")
	_, err = file.WriteString(`{
  "vault": {
    "address": "https://vault.domain.com",
    "token": "root",
    "database": {
      "path": "mount/creds",
      "role": "role"
    },
    "transit": {
      "name": "transit-mount",
      "key": "key-name"
    },
    "kvv2_mount": "kvv2-mount",
    "keydb": {
      "name": "keydb"
    }
  }
}`)
	require.NoError(t, err, "expected no error when writing to config file")

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&configWatcherCalled) == 1
	}, 3*time.Second, 100*time.Millisecond, "expected config watcher to be called after config change")
}
