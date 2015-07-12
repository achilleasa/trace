package trace_test

import (
	"testing"
	"time"

	"github.com/achilleasa/trace"
	"github.com/achilleasa/trace/storage"
)

func TestCollector(t *testing.T) {
	storage := storage.NewMemory()
	collector := trace.NewCollector(storage, 1000, time.Hour)

	stored := make(chan struct{})
	storage.AfterStore = func() {
		stored <- struct{}{}
	}

	// emit trace
	now := time.Now()
	traceId := "abcd-1234-1234-1234"
	collector.TraceChan <- trace.Record{
		Type:      trace.Response,
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
	storage := storage.NewMemory()
	collector := trace.NewCollector(storage, 1000, time.Hour)

	stored := make(chan struct{})
	storage.AfterStore = func() {
		stored <- struct{}{}
	}

	// emit trace
	now := time.Now()
	traceId := "abcd-1234-1234-1234"
	collector.TraceChan <- trace.Record{
		Type:      trace.Response,
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
