package trace

import (
	"time"

	"sort"

	"github.com/achilleasa/usrv/middleware"
)

// This storage backend stores data in memory. It is meant to be used for running tests.
// The backend does not support TTL on keys.
type MemoryStorage struct {
	traces      map[string]middleware.Trace
	services    map[string]string
	serviceDeps map[string]*ServiceDependencies

	// A function invoked after a log entry is stored.
	afterStore func()
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		traces:      make(map[string]middleware.Trace),
		services:    make(map[string]string),
		serviceDeps: make(map[string]*ServiceDependencies),
		afterStore:  func() {},
	}
}

// Store a trace entry and set a TTL on it. If the ttl is 0 then the
// trace record will never expire. Implements the Storage interface.
func (s *MemoryStorage) Store(logEntry *middleware.TraceEntry, ttl time.Duration) error {
	_, exists := s.traces[logEntry.TraceId]
	if !exists {
		s.traces[logEntry.TraceId] = make(middleware.Trace, 0)
	}
	s.traces[logEntry.TraceId] = append(s.traces[logEntry.TraceId], *logEntry)

	s.services[logEntry.From] = logEntry.From
	if logEntry.Type == middleware.Request {
		_, exists = s.serviceDeps[logEntry.From]
		if !exists {
			s.serviceDeps[logEntry.From] = &ServiceDependencies{
				Service:      logEntry.From,
				Dependencies: make([]string, 0),
			}
		}
		s.serviceDeps[logEntry.From].Dependencies = append(s.serviceDeps[logEntry.From].Dependencies, logEntry.To)
	}

	s.afterStore()
	return nil
}

// Get service dependencies optionally filtered by a set of service names. If no filters are
// specified then the response will include all services currently known to the storage.
func (s *MemoryStorage) GetDependencies(srvFilter ...string) ([]ServiceDependencies, error) {
	if len(srvFilter) == 0 {
		srvFilter = make([]string, 0)
		for _, srvName := range s.services {
			srvFilter = append(srvFilter, srvName)
		}
	}

	// Sort service names alphabetically
	sort.Strings(srvFilter)

	replyCount := len(srvFilter)
	serviceDeps := make([]ServiceDependencies, replyCount)
	for index, srvName := range srvFilter {
		dep, exists := s.serviceDeps[srvName]
		if !exists {
			dep = &ServiceDependencies{
				Service:      srvName,
				Dependencies: make([]string, 0),
			}
		}
		serviceDeps[index] = *dep
	}

	return serviceDeps, nil

}

// Fetch a set of time-ordered trace entries with the given trace-id.
func (s *MemoryStorage) GetTrace(traceId string) (middleware.Trace, error) {
	trace, exists := s.traces[traceId]
	if !exists {
		return make(middleware.Trace, 0), nil
	}

	sort.Sort(trace)

	return trace, nil
}

// Shutdown the storage.
func (s *MemoryStorage) Close() error {
	return nil
}
