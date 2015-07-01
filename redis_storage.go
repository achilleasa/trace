package trace

import (
	"time"

	"encoding/json"
	"fmt"

	"sort"

	"github.com/garyburd/redigo/redis"
)

// This storage backend is built on top of Redis. Internally it uses
// a connection pool to provide thread-safe access.
type Redis struct {
	connPool *redis.Pool
}

// Create a new Redis storage.
func NewRedisStorage(redisEndpoint string, password string, db uint, timeout time.Duration) *Redis {
	return &Redis{
		connPool: &redis.Pool{
			MaxIdle:     3,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				c, err := redis.DialTimeout("tcp", redisEndpoint, timeout, timeout, timeout)
				if err != nil {
					return nil, err
				}
				if password != "" {
					if _, err = c.Do("AUTH", password); err != nil {
						c.Close()
						return nil, err
					}
				}
				if db > 0 {
					if _, err = c.Do("SELECT", db); err != nil {
						c.Close()
						return nil, err
					}
				}

				return c, err
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				return err
			},
		},
	}
}

// Store a trace entry and set a TTL on it. If the ttl is 0 then the
// trace record will never expire. Implements the Storage interface.
func (r *Redis) Store(logEntry *TraceEntry, ttl time.Duration) error {
	json, err := json.Marshal(logEntry)
	if err != nil {
		return err
	}

	conn := r.connPool.Get()
	defer conn.Close()

	conn.Send("MULTI")

	// Append log entry to a list that shares the same traceId
	// and set a TTL
	traceKey := fmt.Sprintf("trace.%s", logEntry.TraceId)
	conn.Send("LPUSH", traceKey, json)
	if ttl > time.Second {
		conn.Send("EXPIRE", traceKey, ttl.Seconds())
	}

	// Add logEntry.From to the set of known services
	conn.Send("SADD", "trace.services", logEntry.From)

	// If this is an outgoing request, add the destination to the dependency set
	// for the origin
	if logEntry.Type == Request {
		conn.Send("SADD", fmt.Sprintf("trace.%s.deps", logEntry.From), logEntry.To)
	}

	// Exec pipeline
	_, err = conn.Do("EXEC")
	return err
}

// Fetch a set of time-ordered trace entries with the given trace-id.
func (r *Redis) GetTrace(traceId string) (Trace, error) {

	conn := r.connPool.Get()
	defer conn.Close()

	// Get the number of records
	traceKey := fmt.Sprintf("trace.%s", traceId)
	len, err := redis.Int(conn.Do("LLEN", traceKey))
	if err != nil {
		return nil, err
	}

	// Fetch all records
	rawRows, err := redis.Strings(conn.Do("LRANGE", traceKey, 0, len))
	if err != nil {
		return nil, err
	}

	// Unmarshal raw data
	trace := make(Trace, len)
	for index, rawRow := range rawRows {
		entry := TraceEntry{}
		err = json.Unmarshal([]byte(rawRow), &entry)
		if err != nil {
			return nil, err
		}
		trace[index] = entry
	}

	// Sort the trace so entries appear in insertion order
	sort.Sort(trace)

	return trace, nil
}

// Get service dependencies optionally filtered by a set of service names. If no filters are
// specified then the response will include all services currently known to the storage.
func (r *Redis) GetDependencies(srvFilter ...string) ([]ServiceDependencies, error) {
	conn := r.connPool.Get()
	defer conn.Close()

	var err error

	if len(srvFilter) == 0 {
		srvFilter, err = redis.Strings(conn.Do("SMEMBERS", "trace.services"))
		if err != nil {
			return nil, err
		}
	}

	// Fetch deps in a single batch
	conn.Send("MULTI")
	for _, serviceName := range srvFilter {
		conn.Send("SMEMBERS", fmt.Sprintf("trace.%s.deps", serviceName))
	}
	replies, err := redis.Values(conn.Do("EXEC"))
	if err != nil {
		return nil, err
	}

	// Assemble deps
	replyCount := len(srvFilter)
	serviceDeps := make([]ServiceDependencies, replyCount)
	for index := 0; index < replyCount; index++ {
		deps, _ := redis.Strings(replies[index], nil)
		serviceDeps[index] = ServiceDependencies{
			Service:      srvFilter[index],
			Dependencies: deps,
		}
	}

	return serviceDeps, nil
}

// Shutdown the storage.
func (r *Redis) Close() error {
	return r.connPool.Close()
}
