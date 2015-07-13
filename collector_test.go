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

	wait := make(chan struct{})
	collector.OnTraceAdded = func(rec *trace.Record) {
		// Signal the test we are handling the record
		wait <- struct{}{}
	}

	// emit trace
	now := time.Now()
	traceId := "abcd-1234-1234-1234"
	collector.Add(&trace.Record{
		Type:      trace.Response,
		From:      "com.service3",
		To:        "com.service2",
		Timestamp: now,
		TraceId:   traceId,
	})

	// wait for trace to be processed
	select {
	case <-wait:
	case <-time.After(time.Second * 5):
		t.Fatalf("trace was not handled after 5 sec")
	}
}

func TestCollectorChannelClose(t *testing.T) {
	storage := storage.NewMemory()
	collector := trace.NewCollector(storage, 1000, time.Hour)

	wait := make(chan struct{})
	collector.OnTraceAdded = func(rec *trace.Record) {
		// Signal the test we are handling the record
		wait <- struct{}{}
	}

	// emit trace
	now := time.Now()
	traceId := "abcd-1234-1234-1234"
	collector.Add(&trace.Record{
		Type:      trace.Response,
		From:      "com.service3",
		To:        "com.service2",
		Timestamp: now,
		TraceId:   traceId,
	})

	// wait for trace to be processed
	select {
	case <-wait:
	case <-time.After(time.Second * 5):
		t.Fatalf("trace was not handled after 5 sec")
	}
}

func TestCollectorQueueOverrun(t *testing.T) {
	storage := storage.NewMemory()
	collector := trace.NewCollector(storage, 1, time.Hour)

	wait := make(chan struct{})
	collector.OnTraceAdded = func(rec *trace.Record) {
		// Signal the test we are handling the record
		wait <- struct{}{}

		// Wait for the test to unblock us
		<-wait
	}

	// emit trace
	now := time.Now()
	traceId := "abcd-1234-1234-1234"
	rec := &trace.Record{
		Type:      trace.Response,
		From:      "com.service3",
		To:        "com.service2",
		Timestamp: now,
		TraceId:   traceId,
	}

	addOk := collector.Add(rec)
	if addOk != true {
		t.Fatalf("Expected trace to be successfully queued")
	}

	// After this point we are handling the first trace
	<-wait

	// emit second trace; this should be rejected (queue is full as we are still processing the first trace)
	addOk = collector.Add(rec)
	if addOk != false {
		t.Fatalf("Expected trace to be rejected")
	}

	// Unblock trace 1
	wait <- struct{}{}
}
