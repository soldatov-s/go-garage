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
* http clients pool
* service meta informations
* wrapper for cli based on [spf13/cobra](github.com/spf13/cobra)
* utils for strings and time
* null types (nullmeta, nullstring, nulltime)

# How it works
Databases, cache, metrics, config parsers, logger, http servers all of it is providers in go-garage.
All providers stored in special object `Providers` in contex.Context.  
For user code used domains, they stored in special object `Domains` in contex.Context.  
When you initilize your service you must add all providers and domains in context (funcs with prefix Registrate) after it you can get any object from context (funcs with prefix Get).  
All providers have config structures which can be automated parsed by config parser for getting parameters from environment.  
The framework supports swagger specifiation. In each REST API handler you can add swagger description via [soldatov-s/go-swagger](https://github.com/soldatov-s/go-swagger)

## Example
You can find example of using go-garage in [go-garage-example](https://github.com/soldatov-s/go-garage-example)