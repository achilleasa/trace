package trace

import (
	"testing"
	"time"

	"github.com/achilleasa/usrv/middleware"
)

func TestCollector(t *testing.T) {
	storage := NewMemoryStorage()
	collector := NewCollector(storage)

	stored := make(chan struct{})
	storage.afterStore = func() {
		stored <- struct{}{}
	}

	// emit trace
	now := time.Now()
	traceId := "abcd-1234-1234-1234"
	collector.TraceChan <- middleware.TraceEntry{
		Type:      middleware.Response,
		From:      "com.service3",
		To:        "com.service2",
		Timestamp: now,
		TraceId:   traceId,
	}

	// wait for trace to be processed
	select {
	case <-stored:
	case <-time.After(time.Second * 5):
		t.Fatalf("trace was not handled after 5 sec")
	}

	// Close collector
	collector.Close()
}

func TestCollectorChannelClose(t *testing.T) {
	storage := NewMemoryStorage()
	collector := NewCollector(storage)

	stored := make(chan struct{})
	storage.afterStore = func() {
		stored <- struct{}{}
	}

	// emit trace
	now := time.Now()
	traceId := "abcd-1234-1234-1234"
	collector.TraceChan <- middleware.TraceEntry{
		Type:      middleware.Response,
		From:      "com.service3",
		To:        "com.service2",
		Timestamp: now,
		TraceId:   traceId,
	}

	// wait for trace to be processed
	select {
	case <-stored:
	case <-time.After(time.Second * 5):
		t.Fatalf("trace was not handled after 5 sec")
	}

	// Close trace channel
	close(collector.TraceChan)

	// Allow thread to exit
	<-time.After(time.Millisecond)

	// Close collector
	collector.Close()

	// 2nd close attempt should be no-op
	collector.Close()
}
