package tracer

import "time"

type Collector struct {
	// A set of tokens for bounding the number of concurrent trace records that can be handled
	tokens chan struct{}

	// The storage engine for the processed data
	Storage Storage

	// A TTL for removing trace entries. A value of 0 indicates no TTL.
	tracettl time.Duration

	// This method is invoked when a trace is received. This method is
	// stubbed when running unit tests.
	OnTraceAdded func(rec *Record)
}

// Create a new collector using the supplied storage and allocate a processing queue with depth equal
// to queueSize. The queueSize parameter should be large enough to handle the rate at which your
// service emits trace events.
func NewCollector(storage Storage, queueSize int, tracettl time.Duration) (*Collector, error) {
	collector := &Collector{
		tokens:   make(chan struct{}, queueSize),
		Storage:  storage,
		tracettl: tracettl,
	}

	// Add initial tokens
	for i := 0; i < queueSize; i++ {
		collector.tokens <- struct{}{}
	}

	return collector, storage.Dial()
}

// Append a trace entry. If the collector trace queue is full then the entry will be
// discarded. The method returns true if the trace was successfully enqueued, false otherwise.
func (c *Collector) Add(rec *Record) bool {
	select {
	case token := <-c.tokens:
		go func() {
			defer func() {
				c.tokens <- token
			}()

			c.Storage.Store(rec, c.tracettl)
			if c.OnTraceAdded != nil {
				c.OnTraceAdded(rec)
			}
		}()
		return true
	default:
		// channel is full, discard trace
		return false
	}
}
