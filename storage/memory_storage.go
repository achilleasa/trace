package storage

import (
	"time"

	"sort"

	"sync"

	"github.com/achilleasa/usrv-tracer"
)

// This storage backend stores data in memory. It is meant to be used for running tests.
// The backend does not support TTL on keys.
type Memory struct {
	sync.Mutex
	traces      map[string]tracer.Trace
	services    map[string]string
	serviceDeps map[string]*tracer.Dependencies
	AfterStore  func()
}

func NewMemory() *Memory {
	return &Memory{
		traces:      make(map[string]tracer.Trace),
		services:    make(map[string]string),
		serviceDeps: make(map[string]*tracer.Dependencies),
	}
}

// Store a trace entry and set a TTL on it. If the ttl is 0 then the
// trace record will never expire. Implements the Storage interface.
func (s *Memory) Store(logEntry *tracer.Record, ttl time.Duration) error {
	s.Lock()
	defer s.Unlock()

	_, exists := s.traces[logEntry.TraceId]
	if !exists {
		s.traces[logEntry.TraceId] = make(tracer.Trace, 0)
	}
	s.traces[logEntry.TraceId] = append(s.traces[logEntry.TraceId], *logEntry)

	s.services[logEntry.From] = logEntry.From
	if logEntry.Type == tracer.Request {
		_, exists = s.serviceDeps[logEntry.From]
		if !exists {
			s.serviceDeps[logEntry.From] = &tracer.Dependencies{
				Service:      logEntry.From,
				Dependencies: make([]string, 0),
			}
		}
		// Append dependency if new
		exists = false
		for _, srvName := range s.serviceDeps[logEntry.From].Dependencies {
			if srvName == logEntry.To {
				exists = true
				break
			}
		}
		if !exists {
			s.serviceDeps[logEntry.From].Dependencies = append(s.serviceDeps[logEntry.From].Dependencies, logEntry.To)
		}
	}
	if s.AfterStore != nil {
		s.AfterStore()
	}
	return nil
}

// Get service dependencies optionally filtered by a set of service names. If no filters are
// specified then the response will include all services currently known to the storage.
func (s *Memory) GetDependencies(srvFilter ...string) ([]tracer.Dependencies, error) {
	s.Lock()
	defer s.Unlock()

	if len(srvFilter) == 0 {
		srvFilter = make([]string, 0)
		for _, srvName := range s.services {
			srvFilter = append(srvFilter, srvName)
		}
	}

	// Sort service names alphabetically
	sort.Strings(srvFilter)

	replyCount := len(srvFilter)
	serviceDeps := make([]tracer.Dependencies, replyCount)
	for index, srvName := range srvFilter {
		dep, exists := s.serviceDeps[srvName]
		if !exists {
			dep = &tracer.Dependencies{
				Service:      srvName,
				Dependencies: make([]string, 0),
			}
		}
		serviceDeps[index] = *dep
	}

	return serviceDeps, nil

}

// Fetch a set of time-ordered trace entries with the given trace-id.
func (s *Memory) GetTrace(traceId string) (tracer.Trace, error) {
	s.Lock()
	defer s.Unlock()

	traceLog, exists := s.traces[traceId]
	if !exists {
		return make(tracer.Trace, 0), nil
	}

	sort.Sort(traceLog)

	return traceLog, nil
}

// Shutdown the storage.
func (s *Memory) Close() {
}
