package trace

import (
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestTraceSorting(t *testing.T) {
	now := time.Now()

	type spec struct {
		input    Trace
		expected Trace
	}

	testCases := []spec{
		{
			input: Trace{
				TraceEntry{Type: Request, Timestamp: now.Add(time.Second * 2)},
				TraceEntry{Type: Response, Timestamp: now.Add(time.Second)},
				TraceEntry{Type: Request, Timestamp: now},
			},
			expected: Trace{
				TraceEntry{Type: Request, Timestamp: now},
				TraceEntry{Type: Response, Timestamp: now.Add(time.Second)},
				TraceEntry{Type: Request, Timestamp: now.Add(time.Second * 2)},
			},
		},
		{
			input: Trace{
				TraceEntry{Type: Response, Timestamp: now},
				TraceEntry{Type: Request, Timestamp: now},
			},
			expected: Trace{
				TraceEntry{Type: Request, Timestamp: now},
				TraceEntry{Type: Response, Timestamp: now},
			},
		},
		{
			input: Trace{
				TraceEntry{Type: Request, Timestamp: now},
				TraceEntry{Type: Response, Timestamp: now},
			},
			expected: Trace{
				TraceEntry{Type: Request, Timestamp: now},
				TraceEntry{Type: Response, Timestamp: now},
			},
		},
		{
			input: Trace{
				TraceEntry{Type: Request, Timestamp: now},
				TraceEntry{Type: Response, Timestamp: now.Add(time.Second)},
			},
			expected: Trace{
				TraceEntry{Type: Request, Timestamp: now},
				TraceEntry{Type: Response, Timestamp: now.Add(time.Second)},
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
