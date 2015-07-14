# trace
[![Build Status](https://drone.io/github.com/achilleasa/trace/status.png)](https://drone.io/github.com/achilleasa/trace/latest)
[![codecov.io](http://codecov.io/github/achilleasa/usrv/coverage.svg?branch=master)](http://codecov.io/github/achilleasa/trace?branch=master)

Microservice request/response trace collection and visualization.

The trace package provides an asynchronous request trace collection and storage framework. The collection 
framework may be used standalone or together with the [usrv](https://github.com/achilleasa/usrv) package
with the provided [middleware](https://github.com/achilleasa/trace/tree/master/middleware).

# Dependencies

When using redis as the trace storage engine:

```
go get github.com/achilleasa/service-adapters
go get github.com/garyburd/redigo/redis
```

If you plan on using etcd for automatically configuring your storage engine you also need the following:
```
go get github.com/coreos/go-etcd/...
go get github.com/ugorji/go/codec
```

If using the [usrv](https://github.com/achilleasa/usrv) middleware you also need:
```
go get "code.google.com/p/go-uuid/uuid"
```

# Trace collector

The trace collector uses a `token-based` scheme for processing in `non-blocking mode` trace entries added by an application 
via the collector's `Add` method. For each received trace, a separate go-routine is 
spawned to append the trace to the storage engine that was specified during the collector's construction.

## Queue size
The trace collection process must run in such a way so as not to affect the running application (e.g introduct lag or
block). The queue size constructor option defines a bound on the number of trace entries that can be processed in parallel
by the collector. 

To ensure non-blocking semantics, the collector will `discard` incoming trace records if the
processing queue is full. Consequently, the application should tune the queue size parameter depending on the
estimated trace record througput.

## Trace TTL

You may specify a trace TTL when creating the collector. This ensures a bound on the total number of trace records that are retained by the underlying storage engine. Trace TTL values are specified as `time.Duration` objects.

A value of 0 will effectively disable the TTL for trace records.

## Storage

Storage engines record the incoming trace logs as well as maintain a list of dependencies between services. The
service dependency list is built lazily as trace logs are processed by the collector.

### Redis storage

The redis storage engine builds on top of the redis service adapter offered by the `github.com/achilleasa/service-adapters`
package (see [dependencies](#dependencies)). It supports TTL values expressed in seconds (TTL values < 1 sec will be ingored).

### Memory storage

The memory storage engine is mainly used for testing. It stores data in memory and offers no support for trace log TTL (any specified TTL value will be ignored). It is not recommended to use this storage in production.

### Other storage engines

You can create storage engines for your favorite backend by implementing the [Storage](https://github.com/achilleasa/trace/blob/master/storage.go) interface.

# Usrv request tracer middleware

If you are using the [usrv](https://github.com/achilleasa/usrv) package you can easily add tracing support by
adding the provided middleware when you define your service endpoint. 

All usrv requests that are processed by the middleware
will inject the `middleware.CtxTraceId` header into the outgoing request messages. Its value is a UUID that 
allows the storage service to group together trace records that belong to the same request. The same value will
also be injected into the `context` that gets passed to the service endpoint handler.

A very common scenario is that a microservice will invoke several other microservices (sequentially or in parallel). 
When the `middleware` sub-package is included it will, as a side-effect, patch all usrv client instances so that they also include the `middleware.CtxTraceId` as long as it is present in the `context` that gets passed to the client `Request` and
`RequestWithTimeout` methods.

A basic example (with crude serialization and no error checking) that illustrates how the tracer middleware works
is available [here](https://github.com/achilleasa/trace/blob/master/example/example.go). The example declares
two microservice endpoints:
- the `add/2` service that adds 2 numbers `a, b` and returns the result `a + b`
- the `add/4` service that adds 4 numbers `a, b, c, d` by invoking (in parallel) `add/2` for `a, b` and `c, d` and then `add/2` again on the partial sums to calculate the total sum

The `main` function pretends to be the `com.test.api` endpoint, invokes the `add/4` service froand then logs the request trace and service dependencies. When running the example you will get an output like the following:

```
go run example/example.go

[5413ad95-7b44-4ffd-804e-bacbefae2f9a] Sum: 1 + 3 + 5 + 7 = 16

Trace log:
   {2015-07-13 23:15:15.235346827 +0300 EEST 5413ad95-7b44-4ffd-804e-bacbefae2f9a 02376452-2c22-4cd9-8b58-5eeade37c3d8 REQ com.test.api com.test.add/4 arakis 0 }
   {2015-07-13 23:15:15.235446373 +0300 EEST 5413ad95-7b44-4ffd-804e-bacbefae2f9a afc4c75b-7562-4fba-8df4-360346a73175 REQ com.test.add/4 com.test.add/2 arakis 0 }
   {2015-07-13 23:15:15.235484866 +0300 EEST 5413ad95-7b44-4ffd-804e-bacbefae2f9a f28e223b-984b-4c1e-a9f3-24dfdd5a687f REQ com.test.add/4 com.test.add/2 arakis 0 }
   {2015-07-13 23:15:15.236674233 +0300 EEST 5413ad95-7b44-4ffd-804e-bacbefae2f9a afc4c75b-7562-4fba-8df4-360346a73175 RES com.test.add/2 com.test.add/4 arakis 1226606 }
   {2015-07-13 23:15:15.237729757 +0300 EEST 5413ad95-7b44-4ffd-804e-bacbefae2f9a f28e223b-984b-4c1e-a9f3-24dfdd5a687f RES com.test.add/2 com.test.add/4 arakis 2243808 }
   {2015-07-13 23:15:15.237791668 +0300 EEST 5413ad95-7b44-4ffd-804e-bacbefae2f9a 5655d580-f529-4513-8e9e-63d29905f3fe REQ com.test.add/4 com.test.add/2 arakis 0 }
   {2015-07-13 23:15:15.240205945 +0300 EEST 5413ad95-7b44-4ffd-804e-bacbefae2f9a 5655d580-f529-4513-8e9e-63d29905f3fe RES com.test.add/2 com.test.add/4 arakis 2413169 }
   {2015-07-13 23:15:15.240230707 +0300 EEST 5413ad95-7b44-4ffd-804e-bacbefae2f9a 02376452-2c22-4cd9-8b58-5eeade37c3d8 RES com.test.add/4 com.test.api arakis 4880111 }

Service dependencies:
   [com.test.add/2] depends on: []
   [com.test.add/4] depends on: [com.test.add/2]
   [com.test.api] depends on: [com.test.add/4]
```

# Request visualization web-app

The package ships with a mini angular-js web-app that can be used for visualizing request traces and
figuring out service dependencies. 

To start the web app with default settings (redis at localhot:6379 and app on http://localhost:8080) use the following command:

`go run http/main.go`

A list of supported command line arguments is available by invoking the above command with `-h` or `--help`:
```
go run http/main.go -h

Usage:
  -etcd-hosts="": Etcd host list. If defined, etcd will be used for retrieving redis configuration
  -port=8080: The http server port
  -redis-db=0: Redis db number
  -redis-host=":6379": Redis host (including port)
  -redis-password="": Redis password
```

After the app starts point your browser to [http://localhost:8080](http://localhost:8080) to access the trace visualization UI.

## View request sequence diagram

The sequence diagram view renders a UML sequence diagram for a particular request given its traceId. 

The diagram:
- includes roundtrip times for each call and for the entire request.
- indicates errors (timeouts e.t.c) with a different line type.

![request sequence diagram](https://drive.google.com/uc?export=&id=0Bz9Vk3E_v2HBa1hyS09VNUlGdzg)

## Service dependency visualization

The service dependency graph queries the collector's storage engine for a list of all known services and their direct
dependencies and renders two chart types:
- service dependency chart with all known dependencies and their relations
- the entire dependency tree (direct and indirect service dependencies) for a selected service

### Service dependency chart

The service dependency chart is an interactive circular D3 plot that uses [Danny Holten's](http://www.win.tue.nl/~dholten/) hierarchical edge bundling algorithm. By hovering over a specific service the chart will color:
- services which directly depend on the hovered service in red
- direct dependencies of the hovered service in green

![dependency chart](https://drive.google.com/uc?export=&id=0Bz9Vk3E_v2HBZTMyN0tyWG1OLWM)

### Direct and indirect dependencies of a service

By clicking on a dependency chart service, the view will switch to a filtered mode displaying a tree-like view with the direct and indirect dependencies of the selected service.

![dependency tree](https://drive.google.com/uc?export=&id=0Bz9Vk3E_v2HBSmdtSFVYRUNxVkk)

# License

trace is distributed under the [MIT license](https://github.com/achilleasa/trace/blob/master/LICENSE).
