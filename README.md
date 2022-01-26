[![PkgGoDev](https://pkg.go.dev/badge/github.com/soldatov-s/go-garage?status.svg)](https://pkg.go.dev/github.com/soldatov-s/go-garage)
[![Go](https://github.com/soldatov-s/go-garage/actions/workflows/go.yml/badge.svg)](https://github.com/soldatov-s/go-garage/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/soldatov-s/go-garage/branch/main/graph/badge.svg?token=cPIqyFkCtO)](https://codecov.io/gh/soldatov-s/go-garage)
[![codebeat badge](https://codebeat.co/badges/d737dbca-7067-4d62-84a3-8f0df0b8958a)](https://codebeat.co/projects/github-com-soldatov-s-go-garage-main)
[![Go Report Card](https://goreportcard.com/badge/github.com/soldatov-s/go-garage)](https://goreportcard.com/report/github.com/soldatov-s/go-garage)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

# go-garage
What is a garage? It is place where we store many useful tools which may be needed in the future.
And this framework like a garage. It contains many useful tools/things for fast creating microservice 
with http-servers, swagger, different databases, message brokers, metrics, logger.

## Purposes
For fast creating simple microservices, without designing architecture, with simple config.

## Features
go-garage supports:  
* databases:
  * postgresql (with migrations based on [pressly/goose](https://github.com/pressly/goose))
  * clickhouse (with migrations based on [pressly/goose](https://github.com/pressly/goose))
  * mongodb
  * dbgraph
* message broker:
  * rabbitmq with autoreconect on fail connection
* caches:
  * redis
* opcua:
  * gopcua (gopcua/opcua)[https://github.com/gopcua/opcua]
* metrics (each may be extended in user service):
  * alive handler
  * ready handler (includes communicate with databases and cache)
  * prometheus metrics (includes communicate with databases and cache, current cache size, database metrics from sql.DBStats)
* config parser based on [vrischmann/envconfig](github.com/vrischmann/envconfig)
* http servers (with swagger):
  * router (labstack/echo)(github.com/labstack/echo)
* logger based (rs/zerolog)[github.com/rs/zerolog] 
* dictributed mutex:
  * based on postgresql
  * based on redis
* miniorm for popular actions on the database (simple CRUD)
* service meta informations
* utils for strings (also email, phones) and time
* null types (nullmeta, nullstring, nulltime)

# How works with providers
In common case enough few simple steps:
* create config
* fill config from envs
* pass config to provider constructor
* start provider
* shutdown provider
Also you can get a metrics and handlers for checking that provider alive and ready.  
Consider an example based on a PostgreSQL provider
## Create and fill config
go-garage uses the github.com/vrischmann/envconfig for filling the config envs.
But go-garage allows to set a default values for config.
```go
type Config struct {
	DB          *pq.Config
}

func NewConfig() (*Config, error) {
	c := &Config{
		DB: &pq.Config{
			DSN: "postgres://postgres:secret@postgres:5432/test",
			Migrate: &migrations.Config{
				Directory: "/internal/db/migrations/pg",
				Action:    "up",
			},
		},
	}

	if err := envconfig.Init(c); err != nil {
		return nil, errors.Wrap(err, "init config")
	}

	return c, nil
}

```
## Pass config to provider constructor

```go
// Create connection to PostgreSQL
pqEnity, err := pq.NewEnity(ctx, "garage_pq", config.DB)
if err != nil {
	return errors.Wrap(err, "pq new enity")
}
```
## Start provider
```go
if err := pqEnity.Start(ctx, errGroup); err != nil {
	return errors.Wrap(err, "start")
}
```
## Shutdown provider
```go
if err := pqEnity.Shutdown(ctx); err != nil {
	return errors.Wrap(err, "shutdown")
}
```
## Metrics and alive/ready handlers
```go
pqEnity.GetMetrics() // get metrics
pqEnity.GetAliveHandlers() // alive handlers
pqEnity.GetReadyHandlers() // ready handlers
```

## Example
[go-garage-example](https://github.com/soldatov-s/go-garage-example)