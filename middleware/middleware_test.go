package middleware

import (
	"errors"
	"testing"

	"time"

	"github.com/achilleasa/usrv"
	tracePkg "github.com/achilleasa/usrv-tracer"
	"github.com/achilleasa/usrv-tracer/storage"
	"github.com/achilleasa/usrv/usrvtest"
	"golang.org/x/net/context"
)

func TestTracerWithoutTraceId(t *testing.T) {
	var err error

	processedChan := make(chan struct{})

	storage := storage.Memory
	defer storage.Close()
	storage.AfterStore(func() {
		processedChan <- struct{}{}
	})

	collector, err := tracePkg.NewCollector(storage, 1000, time.Hour)
	if err != nil {
		t.Fatalf("Error creating collector: %v", err)
	}

	ep := usrv.Endpoint{
		Name: "traceTest",
		Handler: usrv.HandlerFunc(func(ctx context.Context, rw usrv.ResponseWriter, req *usrv.Message) {
		}),
	}

	err = Tracer(collector)(&ep)
	if err != nil {
		t.Fatalf("Error applying Tracer() to endpoint: %v", err)
	}

	msg := &usrv.Message{
		From:          "sender",
		To:            "recipient",
		CorrelationId: "123",
	}

	// Send a request without a trace id
	w := usrvtest.NewRecorder()
	ep.Handler.Serve(context.Background(), w, msg)

	traceId := w.Header().Get(CtxTraceId)
	if traceId == nil {
		t.Fatalf("Expected middleware to set response writer header %s", CtxTraceId)
	}

	// Block till both entries are processed
	<-processedChan
	<-processedChan

	// Fetch trace
	traceLog, err := storage.GetTrace(traceId.(string))
	if err != nil {
		t.Fatalf("Error retrieving trace with id %s: %v", traceId, err)
	}
	if len(traceLog) != 2 {
		t.Fatalf("Expected trace len to be 2; got %d", len(traceLog))
	}

	traceEntryIn := traceLog[0]
	traceEntryOut := traceLog[1]

	// Validate REQ trace
	if traceEntryIn.Type != tracePkg.Request {
		t.Fatalf("Expected trace to be of type %v; got %v", tracePkg.Request, traceEntryIn.Type)
	}
	if traceEntryIn.CorrelationId != msg.CorrelationId {
		t.Fatalf("Expected trace CorrelationId to be %s; got %s", msg.CorrelationId, traceEntryIn.CorrelationId)
	}
	if traceEntryIn.TraceId != traceId {
		t.Fatalf("Expected trace TraceId to be %s; got %s", traceId, traceEntryIn.TraceId)
	}
	if traceEntryIn.Error != "" {
		t.Fatalf("Expected trace Error to be ''; got %v", traceEntryIn.Error)
	}
	if traceEntryIn.From != msg.From {
		t.Fatalf("Expected trace From to be %s; got %s", msg.From, traceEntryIn.From)
	}
	if traceEntryIn.To != msg.To {
		t.Fatalf("Expected trace To to be %s; got %s", msg.To, traceEntryIn.To)
	}

	// Validate RES trace
	if traceEntryOut.Type != tracePkg.Response {
		t.Fatalf("Expected trace to be of type %v; got %v", tracePkg.Response, traceEntryOut.Type)
	}
	if traceEntryOut.CorrelationId != msg.CorrelationId {
		t.Fatalf("Expected trace CorrelationId to be %s; got %s", msg.CorrelationId, traceEntryOut.CorrelationId)
	}
	if traceEntryOut.TraceId != traceId {
		t.Fatalf("Expected trace TraceId to be %s; got %s", traceId, traceEntryOut.TraceId)
	}
	if traceEntryOut.Error != "" {
		t.Fatalf("Expected trace Error to be ''; got %v", traceEntryOut.Error)
	}
	// Out trace should reverse From and To
	if traceEntryOut.From != msg.To {
		t.Fatalf("Expected trace From to be %s; got %s", msg.To, traceEntryOut.From)
	}
	if traceEntryOut.To != msg.From {
		t.Fatalf("Expected trace To to be %s; got %s", msg.From, traceEntryOut.To)
	}

}

func TestTracerWithExistingTraceId(t *testing.T) {
	var err error

	processedChan := make(chan struct{})

	storage := storage.Memory
	defer storage.Close()
	storage.AfterStore(func() {
		processedChan <- struct{}{}
	})

	collector, err := tracePkg.NewCollector(storage, 1000, time.Hour)
	if err != nil {
		t.Fatalf("Error creating collector: %v", err)
	}

	ep := usrv.Endpoint{
		Name: "traceTest",
		Handler: usrv.HandlerFunc(func(ctx context.Context, rw usrv.ResponseWriter, req *usrv.Message) {
		}),
	}

	err = Tracer(collector)(&ep)
	if err != nil {
		t.Fatalf("Error applying Tracer() to endpoint: %v", err)
	}

	msg := &usrv.Message{
		From:          "sender",
		To:            "recipient",
		CorrelationId: "123",
		Headers:       make(usrv.Header),
	}

	// Send a request with an existing trace id
	existingTraceId := "0-0-0-0"
	msg.Headers.Set(CtxTraceId, existingTraceId)

	w := usrvtest.NewRecorder()
	ep.Handler.Serve(context.Background(), w, msg)

	traceId := w.Header().Get(CtxTraceId)
	if traceId == nil {
		t.Fatalf("Expected middleware to set response writer header %s", CtxTraceId)
	}
	if traceId != existingTraceId {
		t.Fatalf("Middleware did not reuse existing traceId %s; got %s", existingTraceId, traceId)
	}

	// Block till both entries are processed
	<-processedChan
	<-processedChan

	// Fetch trace
	traceLog, err := storage.GetTrace(traceId.(string))
	if err != nil {
		t.Fatalf("Error retrieving trace with id %s: %v", traceId, err)
	}
	if len(traceLog) != 2 {
		t.Fatalf("Expected trace len to be 2; got %d", len(traceLog))
	}

	traceEntryIn := traceLog[0]
	traceEntryOut := traceLog[1]

	// Validate REQ trace
	if traceEntryIn.Type != tracePkg.Request {
		t.Fatalf("Expected trace to be of type %v; got %v", tracePkg.Request, traceEntryIn.Type)
	}
	if traceEntryIn.CorrelationId != msg.CorrelationId {
		t.Fatalf("Expected trace CorrelationId to be %s; got %s", msg.CorrelationId, traceEntryIn.CorrelationId)
	}
	if traceEntryIn.TraceId != traceId {
		t.Fatalf("Expected trace TraceId to be %s; got %s", traceId, traceEntryIn.TraceId)
	}
	if traceEntryIn.Error != "" {
		t.Fatalf("Expected trace Error to be ''; got %v", traceEntryIn.Error)
	}
	if traceEntryIn.From != msg.From {
		t.Fatalf("Expected trace From to be %s; got %s", msg.From, traceEntryIn.From)
	}
	if traceEntryIn.To != msg.To {
		t.Fatalf("Expected trace To to be %s; got %s", msg.To, traceEntryIn.To)
	}

	// Validate RES trace
	if traceEntryOut.Type != tracePkg.Response {
		t.Fatalf("Expected trace to be of type %v; got %v", tracePkg.Response, traceEntryOut.Type)
	}
	if traceEntryOut.CorrelationId != msg.CorrelationId {
		t.Fatalf("Expected trace CorrelationId to be %s; got %s", msg.CorrelationId, traceEntryOut.CorrelationId)
	}
	if traceEntryOut.TraceId != traceId {
		t.Fatalf("Expected trace TraceId to be %s; got %s", traceId, traceEntryOut.TraceId)
	}
	if traceEntryOut.Error != "" {
		t.Fatalf("Expected trace Error to be ''; got %v", traceEntryOut.Error)
	}
	// Out trace should reverse From and To
	if traceEntryOut.From != msg.To {
		t.Fatalf("Expected trace From to be %s; got %s", msg.To, traceEntryOut.From)
	}
	if traceEntryOut.To != msg.From {
		t.Fatalf("Expected trace To to be %s; got %s", msg.From, traceEntryOut.To)
	}
}

func TestTracerWithError(t *testing.T) {
	var err error

	processedChan := make(chan struct{})

	storage := storage.Memory
	defer storage.Close()
	storage.AfterStore(func() {
		processedChan <- struct{}{}
	})

	collector, err := tracePkg.NewCollector(storage, 1000, time.Hour)
	if err != nil {
		t.Fatalf("Error creating collector: %v", err)
	}

	ep := usrv.Endpoint{
		Name: "traceTest",
		Handler: usrv.HandlerFunc(func(ctx context.Context, rw usrv.ResponseWriter, req *usrv.Message) {
			rw.WriteError(errors.New("I cannot allow you to do that Dave"))
		}),
	}

	err = Tracer(collector)(&ep)
	if err != nil {
		t.Fatalf("Error applying Tracer() to endpoint: %v", err)
	}

	msg := &usrv.Message{
		From:          "sender",
		To:            "recipient",
		CorrelationId: "123",
	}

	// Send request
	w := usrvtest.NewRecorder()
	ep.Handler.Serve(context.Background(), w, msg)

	traceId := w.Header().Get(CtxTraceId)
	if traceId == nil {
		t.Fatalf("Expected middleware to set response writer header %s", CtxTraceId)
	}

	// Block till both entries are processed
	<-processedChan
	<-processedChan

	// Fetch trace
	traceLog, err := storage.GetTrace(traceId.(string))
	if err != nil {
		t.Fatalf("Error retrieving trace with id %s: %v", traceId, err)
	}
	if len(traceLog) != 2 {
		t.Fatalf("Expected trace len to be 2; got %d", len(traceLog))
	}

	traceEntryOut := traceLog[1]

	// Validate RES trace
	if traceEntryOut.Type != tracePkg.Response {
		t.Fatalf("Expected trace to be of type %v; got %v", tracePkg.Response, traceEntryOut.Type)
	}
	if traceEntryOut.CorrelationId != msg.CorrelationId {
		t.Fatalf("Expected trace CorrelationId to be %s; got %s", msg.CorrelationId, traceEntryOut.CorrelationId)
	}
	if traceEntryOut.TraceId != traceId {
		t.Fatalf("Expected trace TraceId to be %s; got %s", traceId, traceEntryOut.TraceId)
	}
	if traceEntryOut.Error != "I cannot allow you to do that Dave" {
		t.Fatalf("Expected trace Error to be 'I cannot allow you to do that Dave'; got %v", traceEntryOut.Error)
	}
	// Out trace should reverse From and To
	if traceEntryOut.From != msg.To {
		t.Fatalf("Expected trace From to be %s; got %s", msg.To, traceEntryOut.From)
	}
	if traceEntryOut.To != msg.From {
		t.Fatalf("Expected trace To to be %s; got %s", msg.From, traceEntryOut.To)
	}

}
