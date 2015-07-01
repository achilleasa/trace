package trace

import (
	"time"

	"sync"
)

type Collector struct {
	// A channel for triggering a collector shutdown
	shutdownChan chan struct{}

	// A buffered channel for processing trace data
	TraceChan chan TraceEntry

	// A storage engine for the processed data
	storage Storage

	// A waitgroup for ensuring that the collector shuts down properly
	waitGroup sync.WaitGroup
}

// Create a new collector using the supplied storage.
func NewCollector(storage Storage) *Collector {
	collector := &Collector{
		shutdownChan: make(chan struct{}, 1),
		TraceChan:    make(chan TraceEntry, 1000),
		storage:      storage,
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
				go func(evt *TraceEntry) {
					c.storage.Store(evt, time.Hour*1)
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
