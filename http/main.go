package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"time"

	"log"

	"net/http"

	"encoding/json"
	"fmt"

	"github.com/achilleasa/trace"
	"github.com/achilleasa/trace/storage"
)

type server struct {
	storageEngine trace.Storage
}

// Create a new http server for reporting trace and dependecy details.
func newServer(storage trace.Storage) *server {
	return &server{
		storageEngine: storage,
	}
}

// The top-level router for the http server.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handlerFunc := http.NotFound

	if r.Method == "GET" {
		if strings.HasPrefix(r.URL.Path, "/trace/") {
			handlerFunc = s.getTrace
		} else if strings.HasPrefix(r.URL.Path, "/deps/") {
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
	log.Printf("Serving index\n")
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
	srvFilter := strings.Split(r.URL.Query().Get("srv_filter"), ",")

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
	redisEndpoint = flag.String("redis-host", ":6379", "Redis host (including port)")
	redisDb       = flag.Int("redis-db", ":6379", "Redis host (including port)")
	redisPassword = flag.String("redis-password", "", "Redis password")
	port          = flag.Int("port", 8080, "The http server port")
	storageEngine *storage.Redis
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

	flag.Parse()

	log.Printf("Using REDIS storage: %s (using password: %v)\n", *redisEndpoint, *redisPassword != "")
	storageEngine = storage.NewRedis(*redisEndpoint, *redisPassword, 0, time.Second*10)

	log.Printf("Listening for incoming connections on port %d\n", *port)
	http.ListenAndServe(fmt.Sprintf(":%d", *port), newServer(storageEngine))
}
