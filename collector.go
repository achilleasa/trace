package trace

import (
	"time"

	"github.com/achilleasa/usrv/middleware"
)

type Collector struct {
	// A channel for triggering a collector shutdown
	shutdownChan chan struct{}

	// A buffered channel for processing trace data
	TraceChan chan middleware.TraceEntry

	// A storage engine for the processed data
	storage Storage
}

// Create a new collector using the supplied storage.
func NewCollector(storage Storage) *Collector {
	collector := &Collector{
		shutdownChan: make(chan struct{}),
		TraceChan:    make(chan middleware.TraceEntry, 1000),
		storage:      storage,
	}

	collector.start()

	return collector
}

// Start event capturing loop.
func (c *Collector) start() {
	go func() {
		for {
			select {
			case <-c.shutdownChan:
				return
			case evt := <-c.TraceChan:
				go func(evt *middleware.TraceEntry) {
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
	close(c.shutdownChan)
}
