package trace

import (
	"time"

	"sync"
)

type Collector struct {
	// A channel for triggering a collector shutdown
	shutdownChan chan struct{}

	// A buffered channel for processing trace data
	TraceChan chan Record

	// A storage engine for the processed data
	storage Storage

	// A waitgroup for ensuring that the collector shuts down properly
	waitGroup sync.WaitGroup

	// A TTL for removing trace entries. A value of 0 indicates no TTL.
	tracettl time.Duration
}

// Create a new collector using the supplied storage and allocate a processing queue with depth equal
// to queueSize. The queueSize parameter should be large enough to handle the rate at which your
// service emits trace events.
func NewCollector(storage Storage, queueSize int, tracettl time.Duration) *Collector {
	collector := &Collector{
		shutdownChan: make(chan struct{}, 1),
		TraceChan:    make(chan Record, queueSize),
		storage:      storage,
		tracettl:     tracettl,
	}

	collector.start()

	return collector
}

// Start event capturing loop.
func (c *Collector) start() {
	c.waitGroup.Add(1)
	go func() {
		defer c.waitGroup.Done()
		for {
			select {
			case <-c.shutdownChan:
				return
			case evt, ok := <-c.TraceChan:
				if !ok {
					return
				}
				go func(evt *Record) {
					c.storage.Store(evt, c.tracettl)
				}(&evt)
			}
		}
	}()
}

// Shutdown the collector.
func (c *Collector) Close() {
	if c.shutdownChan == nil {
		return
	}

	c.shutdownChan <- struct{}{}
	c.waitGroup.Wait()
	c.shutdownChan = nil
}
