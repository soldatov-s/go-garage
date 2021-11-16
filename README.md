[![PkgGoDev](https://pkg.go.dev/badge/github.com/soldatov-s/go-garage?status.svg)](https://pkg.go.dev/github.com/soldatov-s/go-garage)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

# go-garage
The framework for fast creating microservice with logger, http, db, metrics, queues

## Purposes
For fast creating simple microservices, without designing architecture, with simple config.

## Features
go-garage supports:  
* databases:
  * postgresql (with migrations based on [pressly/goose](https://github.com/pressly/goose))
  * clickhouse (with migrations based on [pressly/goose](https://github.com/pressly/goose))
  * mongodb
  * dbgraph
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
* service meta informations
* utils for strings and time
* null types (nullmeta, nullstring, nulltime)

# How it works
Databases, cache, metrics, config parsers, logger, http servers all of it is providers in go-garage.

## Example
TODO