package trace

import (
	"time"
)

type TraceType string

// The types of traces that are emitted by the Tracer middleware.
const (
	Request  TraceType = "REQ"
	Response TraceType = "RES"
)

// The ServiceDependencies describes a service and its dependencies.
type Dependencies struct {
	Service      string   `json:"service"`
	Dependencies []string `json:"dependencies"`
}

// The TraceEntry structure represents a trace entry
// that is emitted by the Tracer middleware.
type Record struct {
	Timestamp     time.Time `json:"timestamp"`
	TraceId       string    `json:"trace_id"`
	CorrelationId string    `json:"correlation_id"`
	Type          TraceType `json:"type"`
	From          string    `json:"from"`
	To            string    `json:"to"`
	Host          string    `json:"host"`
	Duration      int64     `json:"duration,omitempty"`
	Error         string    `json:"error,omitempty"`
}

// A Trace is a list of TraceLog entries.
type Trace []Record

// Get Trace len. Implements sort.Interface
func (t Trace) Len() int {
	return len(t)
}

// Compare entries. Implements sort.Interface
func (t Trace) Less(l, r int) bool {
	left := t[l]
	right := t[r]

	// Compare by timestamp
	if left.Timestamp.Before(right.Timestamp) {
		return true
	} else if left.Timestamp.After(right.Timestamp) {
		return false
	}

	// If timestamps are equal, prefer Requests over Responses
	if left.Type == Request && right.Type == Response {
		return true
	}

	return false
}

// Swap trace entries. Implements sort.Interface
func (t Trace) Swap(l, r int) {
	t[l], t[r] = t[r], t[l]
}
