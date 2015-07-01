package trace

import (
	"time"
)

// The Storage interface is implemented by providers that can store and query trace data.
type Storage interface {
	// Store a trace entry and set a TTL on it. If the ttl is 0 then the
	// trace record will never expire
	Store(logEntry *Record, ttl time.Duration) error

	// Fetch a set of time-ordered trace entries with the given trace-id
	GetTrace(traceId string) (Trace, error)

	// Get service dependencies optionally filtered by a set of service names. If no filters are
	// specified then the response will include all services currently known to the storage.
	GetDependencies(srvFilter ...string) ([]ServiceDependencies, error)

	// Shutdown the storage.
	Close() error
}
