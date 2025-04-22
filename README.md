# Web

## Description

This repository contains tools that are commonly used between my golang projects. It allows for the projects to be
easily structured and maintained. The tools are designed to be used in a modular way, allowing for easy integration into
existing projects. The repository is easily extensible, allowing for new tools to be added as needed. When creating a
new web app you will need to provide the logger which is accessible from the app at later stages.

## Usage

### Config

You can access the config file using the viper instance from the app. The config file is loaded from the Environment
Variable `CONFIG_LOCATION`; if no location is provided, the app will default to `config.json` in the present directory.
The config file can be in any format that viper supports. Before you can access it you will have to ensure that the app
loads the config with the `WithViperConfig` option. Once the config is loaded you can access it using the `app.Viper()`
method.

### Example

Please see a basic example of how to use the tools in a web app. The example is a simple web app that provides a Vault
client, database connection from Vault and a Http server.

```go
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jacobbrewer1/web"
	"github.com/jacobbrewer1/web/logging"
)

type App struct {
	base *web.App
}

func NewApp(l *slog.Logger) (*App, error) {
	base, err := web.NewApp(l)
	if err != nil {
		return nil, err
	}

	return &App{
		base: base,
	}, nil
}

func handler(l *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.Debug("Received request", "method", r.Method, "url", r.URL.String())
		w.WriteHeader(http.StatusOK)
	})
}

func (a *App) Start() error {
	if err := a.base.Start(
		web.WithViperConfig(),
		web.WithVaultClient(),
		web.WithDatabaseFromVault(),
	); err != nil {
		return fmt.Errorf("start app: %w", err)
	}

	if err := a.base.StartServer("api", &http.Server{
		Addr:              ":8080",
		Handler:           handler(logging.LoggerWithComponent(a.base.Logger(), "api-handler")),
		ReadHeaderTimeout: 5 * time.Second,
	}); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	return nil
}

func (a *App) WaitForEnd() {
	// Wait for the application to finish
	a.base.WaitForEnd(a.Shutdown)
}

func (a *App) Shutdown() {
	a.base.Shutdown()
}

func main() {
	l := logging.NewLogger(
		logging.WithAppName("myapp"),
	)

	app, err := NewApp(l)
	if err != nil {
		panic(err)
	}

	if err := app.Start(); err != nil {
		panic(err)
	}

	app.WaitForEnd()
}
```

The config file for the above example is a simple yaml file that contains the following:

```json
{
  "vault": {
    "address": "http://vault-active.vault:8200",
    "database": {
      "path": "database-mount/creds",
      "role": "vault-database-role"
    }
  },
  "database": {
    "host": "host:port",
    "schema": "schema-name"
  }
}
```

### Deployment

When deploying the application, you will need to ensure that the following environment variables are set for a smooth
operation, this application makes use of the Downward Kubernetes API to set the environment variables.

The following environment variables should be set within the deployment yaml:

```yaml
- name: "SERVICE_ACCOUNT_NAME" # This is used to get the service account name; additionally, Vault will fail to initialize if this is not set.
  valueFrom:
    fieldRef:
      fieldPath: spec.serviceAccountName
- name: "POD_IP" # This is used to get the pod IP address
  valueFrom:
    fieldRef:
      fieldPath: status.podIP
- name: "NODE_NAME" # This is used to get the node name
  valueFrom:
    fieldRef:
      fieldPath: spec.nodeName
```

### Packages

If you require some Kubernetes helpers, you can use the `k8s` package. It contains a few helpers that are useful for
interacting with Kubernetes. Please look at the files within the `k8s` package for more information.
