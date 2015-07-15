package tracer_test

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/achilleasa/usrv-tracer"
)

func TestTraceSorting(t *testing.T) {
	now := time.Now()

	type spec struct {
		input    tracer.Trace
		expected tracer.Trace
	}

	testCases := []spec{
		{
			input: tracer.Trace{
				tracer.Record{Type: tracer.Request, Timestamp: now.Add(time.Second * 2)},
				tracer.Record{Type: tracer.Response, Timestamp: now.Add(time.Second)},
				tracer.Record{Type: tracer.Request, Timestamp: now},
			},
			expected: tracer.Trace{
				tracer.Record{Type: tracer.Request, Timestamp: now},
				tracer.Record{Type: tracer.Response, Timestamp: now.Add(time.Second)},
				tracer.Record{Type: tracer.Request, Timestamp: now.Add(time.Second * 2)},
			},
		},
		{
			input: tracer.Trace{
				tracer.Record{Type: tracer.Response, Timestamp: now},
				tracer.Record{Type: tracer.Request, Timestamp: now},
			},
			expected: tracer.Trace{
				tracer.Record{Type: tracer.Request, Timestamp: now},
				tracer.Record{Type: tracer.Response, Timestamp: now},
			},
		},
		{
			input: tracer.Trace{
				tracer.Record{Type: tracer.Request, Timestamp: now},
				tracer.Record{Type: tracer.Response, Timestamp: now},
			},
			expected: tracer.Trace{
				tracer.Record{Type: tracer.Request, Timestamp: now},
				tracer.Record{Type: tracer.Response, Timestamp: now},
			},
		},
		{
			input: tracer.Trace{
				tracer.Record{Type: tracer.Request, Timestamp: now},
				tracer.Record{Type: tracer.Response, Timestamp: now.Add(time.Second)},
			},
			expected: tracer.Trace{
				tracer.Record{Type: tracer.Request, Timestamp: now},
				tracer.Record{Type: tracer.Response, Timestamp: now.Add(time.Second)},
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
