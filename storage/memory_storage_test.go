package storage

import (
	"testing"
	"time"

	"github.com/achilleasa/trace"

	"reflect"

	"bytes"
	"encoding/json"
	"sort"
)

func TestMemoryStorage(t *testing.T) {
	storage := NewMemory()
	defer storage.Close()

	now := time.Now()
	traceId := "abcd-1234-1234-1234"

	// Shuffled records to simulate appends by different processes
	dataSet := trace.Trace{
		trace.Record{Type: trace.Response, From: "com.service3", To: "com.service2", Timestamp: now.Add(time.Second * 3), TraceId: traceId},
		trace.Record{Type: trace.Request, From: "com.service2", To: "com.service3", Timestamp: now.Add(time.Second * 2), TraceId: traceId},
		trace.Record{Type: trace.Response, From: "com.service2", To: "com.service1", Timestamp: now.Add(time.Second * 4), TraceId: traceId},
		trace.Record{Type: trace.Request, From: "com.service1", To: "com.service2", Timestamp: now.Add(time.Second * 1), TraceId: traceId},
	}

	// Generate the final sorted set that we will use for comparisons
	sortedDataSet := make(trace.Trace, len(dataSet))
	copy(sortedDataSet, dataSet)
	sort.Sort(sortedDataSet)

	// Insert trace
	ttl := time.Minute
	for index, entry := range dataSet {
		err := storage.Store(&entry, ttl)
		if err != nil {
			t.Fatalf("Error while storing entry #%d: %v", index, err)
		}
	}

	// Fetch unknown trace
	traceLog, err := storage.GetTrace("foobar")
	if err != nil {
		t.Fatalf("Error retrieving trace: %v", err)
	}
	if len(traceLog) != 0 {
		t.Fatalf("Expected empty trace; got trace with %d items: %v", len(traceLog), traceLog)
	}

	// Fetch trace by id
	traceLog, err = storage.GetTrace(traceId)
	if err != nil {
		t.Fatalf("Error retrieving trace: %v", err)
	}

	l, _ := json.Marshal(sortedDataSet)
	r, _ := json.Marshal(traceLog)
	if bytes.Compare(l, r) != 0 {
		t.Fatalf("Expected retrieved trace to be equal to %v; got %v", sortedDataSet, traceLog)
	}

	// Get dependencies
	depTests := []trace.Dependencies{
		trace.Dependencies{Service: "com.service1", Dependencies: []string{"com.service2"}},
		trace.Dependencies{Service: "com.service2", Dependencies: []string{"com.service3"}},
		trace.Dependencies{Service: "com.service3", Dependencies: []string{}},
	}

	// Fetch using filters
	for index, depSpec := range depTests {
		deps, err := storage.GetDependencies(depSpec.Service)
		if err != nil {
			t.Fatalf("Error retrieving dep set #%d: %v", index, err)
		}
		if len(deps) != 1 {
			t.Fatalf("Expected retrieved dependencies for set #%d to have length 1; got %d", index, len(deps))
		}
		dep := deps[0]
		if dep.Service != depSpec.Service {
			t.Fatalf("Expected dependency set #%d to contain dependencies for %s; got %s", index, depSpec.Service, dep.Service)
		}
		if !reflect.DeepEqual(depSpec.Dependencies, dep.Dependencies) {
			t.Fatalf("Expected dependency set #%d to contain dependencies %v; got %v", index, depSpec.Dependencies, dep.Dependencies)
		}
	}

	// Fetch all
	deps, err := storage.GetDependencies()
	if err != nil {
		t.Fatalf("Error retrieving dependencies: %v", err)
	}
	if len(depTests) != len(deps) {
		t.Fatalf("Expected retrieved dependencies to have length %d; got %d", len(depTests), len(deps))
	}
	l, _ = json.Marshal(depTests)
	r, _ = json.Marshal(deps)
	if bytes.Compare(l, r) != 0 {
		t.Fatalf("Expected dependency set %v; got %v", depTests, deps)
	}
}
