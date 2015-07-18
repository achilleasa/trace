package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"

	"log"

	"net/http"

	"encoding/json"
	"fmt"

	"strconv"

	"time"

	"github.com/achilleasa/usrv-service-adapters"
	"github.com/achilleasa/usrv-service-adapters/dial"
	"github.com/achilleasa/usrv-service-adapters/service/etcd"
	"github.com/achilleasa/usrv-service-adapters/service/redis"
	"github.com/achilleasa/usrv-tracer"
	"github.com/achilleasa/usrv-tracer/storage"
)

type server struct {
	storageEngine tracer.Storage
}

// Create a new http server for reporting trace and dependency details.
func newServer(storage tracer.Storage) (*server, error) {
	return &server{
		storageEngine: storage,
	}, storage.Dial()
}

// The top-level router for the http server.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handlerFunc := http.NotFound

	if r.Method == "GET" {
		if strings.HasPrefix(r.URL.Path, "/trace/") {
			handlerFunc = s.getTrace
		} else if strings.HasPrefix(r.URL.Path, "/deps") {
			handlerFunc = s.getDeps
		} else if r.URL.Path == "/" {
			handlerFunc = s.getIndex
		}
	}

	// Invoke selected handler
	handlerFunc(w, r)
}

// Serve the index page.
func (s *server) getIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "http/static/index.html")
}

// Get trace by id.
func (s *server) getTrace(w http.ResponseWriter, r *http.Request) {
	// Extract trace id from path and load trace
	traceId := r.URL.Path[7:]
	trace, err := s.storageEngine.GetTrace(traceId)
	if err != nil {
		s.sendError(w, err)
		return
	}

	s.send(w, trace)
}

// Get service dependencies optionally filtered by a list of service names.
func (s *server) getDeps(w http.ResponseWriter, r *http.Request) {
	// Extract filters from GET params
	filterVal := r.URL.Query().Get("srv_filter")
	var srvFilter []string
	if filterVal != "" {
		srvFilter = strings.Split(filterVal, ",")
	} else {
		srvFilter = nil
	}

	trace, err := s.storageEngine.GetDependencies(srvFilter...)
	if err != nil {
		s.sendError(w, err)
		return
	}

	s.send(w, trace)
}

// Report error encoded as json.
func (s *server) sendError(w http.ResponseWriter, err error) {
	data, err := json.Marshal(map[string]string{"error": err.Error()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	w.Write(data)
}

// Report error encoded as json.
func (s *server) send(w http.ResponseWriter, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

var (
	etcdEndpoints = flag.String("etcd-hosts", "", "Etcd host list. If defined, etcd will be used for retrieving redis configuration. You may also specify etcd hosts using the ETCD_HOSTS env var")
	redisEndpoint = flag.String("redis-host", ":6379", "Redis host (including port)")
	redisDb       = flag.Int("redis-db", 0, "Redis db number")
	redisPassword = flag.String("redis-password", "", "Redis password")
	port          = flag.Int("port", 8080, "The http server port")
	storageEngine tracer.Storage
)

func main() {
	// Register shutdown handler
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		for sig := range sigChan {
			log.Printf("Caught %s; shutting down", sig)
			if storageEngine != nil {
				storageEngine.Close()
			}
			os.Exit(0)
		}
	}()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Configure backend adapter
	flag.Parse()

	// Check for etcd config. If present dial etcd adapter
	// If no etcd cmdline arg is set, also check for ETCD_HOSTS env var
	if *etcdEndpoints == "" {
		*etcdEndpoints = os.Getenv("ETCD_HOSTS")
	}
	if *etcdEndpoints != "" {
		etcd.Adapter.SetOptions(
			adapters.Logger(logger),
			adapters.Config(map[string]string{"hosts": *etcdEndpoints}),
		)

		err := etcd.Adapter.Dial()
		if err != nil {
			panic(err)
		}
	}

	// Set redis adapter options
	opts := make([]adapters.ServiceOption, 0)
	opts = append(opts, adapters.Logger(logger))
	opts = append(opts, adapters.DialPolicy(dial.ExpBackoff(10, time.Millisecond)))
	if *etcdEndpoints != "" {
		opts = append(opts, etcd.AutoConf("/config/service/redis"))
	} else {
		opts = append(
			opts,
			adapters.Config(
				map[string]string{
					"endpoint": *redisEndpoint,
					"db":       strconv.Itoa(*redisDb),
					"password": *redisPassword,
				},
			),
		)
	}
	err := redis.Adapter.SetOptions(opts...)
	if err != nil {
		log.Panic(err)
	}

	logger.Printf("[UI-SRV] Listening for incoming connections on port %d; press ctrl+c to exit\n", *port)
	srv, err := newServer(storage.Redis)
	if err != nil {
		log.Panic(err)
	}

	http.ListenAndServe(fmt.Sprintf(":%d", *port), srv)
}
