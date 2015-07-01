package trace_test

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/achilleasa/trace"
)

func TestTraceSorting(t *testing.T) {
	now := time.Now()

	type spec struct {
		input    trace.Trace
		expected trace.Trace
	}

	testCases := []spec{
		{
			input: trace.Trace{
				trace.Record{Type: trace.Request, Timestamp: now.Add(time.Second * 2)},
				trace.Record{Type: trace.Response, Timestamp: now.Add(time.Second)},
				trace.Record{Type: trace.Request, Timestamp: now},
			},
			expected: trace.Trace{
				trace.Record{Type: trace.Request, Timestamp: now},
				trace.Record{Type: trace.Response, Timestamp: now.Add(time.Second)},
				trace.Record{Type: trace.Request, Timestamp: now.Add(time.Second * 2)},
			},
		},
		{
			input: trace.Trace{
				trace.Record{Type: trace.Response, Timestamp: now},
				trace.Record{Type: trace.Request, Timestamp: now},
			},
			expected: trace.Trace{
				trace.Record{Type: trace.Request, Timestamp: now},
				trace.Record{Type: trace.Response, Timestamp: now},
			},
		},
		{
			input: trace.Trace{
				trace.Record{Type: trace.Request, Timestamp: now},
				trace.Record{Type: trace.Response, Timestamp: now},
			},
			expected: trace.Trace{
				trace.Record{Type: trace.Request, Timestamp: now},
				trace.Record{Type: trace.Response, Timestamp: now},
			},
		},
		{
			input: trace.Trace{
				trace.Record{Type: trace.Request, Timestamp: now},
				trace.Record{Type: trace.Response, Timestamp: now.Add(time.Second)},
			},
			expected: trace.Trace{
				trace.Record{Type: trace.Request, Timestamp: now},
				trace.Record{Type: trace.Response, Timestamp: now.Add(time.Second)},
			},
		},
	}

	for index, testCase := range testCases {
		sort.Sort(testCase.input)
		if !reflect.DeepEqual(testCase.expected, testCase.input) {
			t.Fatalf("[case %d] expected sort output to be %v; got %v", index, testCase.expected, testCase.input)
		}
	}
}
