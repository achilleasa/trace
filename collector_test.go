package tracer_test

import (
	"testing"
	"time"

	"github.com/achilleasa/usrv-tracer"
	"github.com/achilleasa/usrv-tracer/storage"
)

func TestCollector(t *testing.T) {
	collector, err := tracer.NewCollector(storage.Memory, 1000, time.Hour)
	if err != nil {
		t.Fatalf("Error creating collector: %v", err)
	}
	defer collector.Storage.Close()

	wait := make(chan struct{})
	collector.OnTraceAdded = func(rec *tracer.Record) {
		// Signal the test we are handling the record
		wait <- struct{}{}
	}

	// emit trace
	now := time.Now()
	traceId := "abcd-1234-1234-1234"
	collector.Add(&tracer.Record{
		Type:      tracer.Response,
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
	collector, err := tracer.NewCollector(storage.Memory, 1000, time.Hour)
	if err != nil {
		t.Fatalf("Error creating collector: %v", err)
	}
	defer collector.Storage.Close()

	wait := make(chan struct{})
	collector.OnTraceAdded = func(rec *tracer.Record) {
		// Signal the test we are handling the record
		wait <- struct{}{}
	}

	// emit trace
	now := time.Now()
	traceId := "abcd-1234-1234-1234"
	collector.Add(&tracer.Record{
		Type:      tracer.Response,
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
	collector, err := tracer.NewCollector(storage.Memory, 1, time.Hour)
	if err != nil {
		t.Fatalf("Error creating collector: %v", err)
	}
	defer collector.Storage.Close()

	wait := make(chan struct{})
	collector.OnTraceAdded = func(rec *tracer.Record) {
		// Signal the test we are handling the record
		wait <- struct{}{}

		// Wait for the test to unblock us
		<-wait
	}

	// emit trace
	now := time.Now()
	traceId := "abcd-1234-1234-1234"
	rec := &tracer.Record{
		Type:      tracer.Response,
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
