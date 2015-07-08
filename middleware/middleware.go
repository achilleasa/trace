package middleware

import (
	"os"
	"time"

	"code.google.com/p/go-uuid/uuid"
	tracePkg "github.com/achilleasa/trace"
	"github.com/achilleasa/usrv"
	"golang.org/x/net/context"
)

var (
	CtxTraceId = "trace_id"
)

func init() {
	// Make sure clients inject the trace-id header in outgoing messages so we can track service dependencies
	usrv.InjectCtxFieldToClients(CtxTraceId)
}

// The tracer middleware emits TraceEntry objects to the supplied Collector whenever the
// server processes an incoming request.
//
// Two Trace entries will be emitted for each request, one for the incoming request
// and one for the outgoing response.
//
// The middleware will inject a traceId for each incoming request into the context
// that gets passed to the request handler. To ensure that any further RPC requests
// that occur inside the wrapped handler are associated with the current request, the
// handler should pass its context to any performed RPC client requests.
//
// This function is designed to emit events in non-blocking mode. If the Collector does
// not have enough capacity to store a generated TraceEntry then it will be silently dropped.
func Tracer(collector *tracePkg.Collector) usrv.EndpointOption {
	return func(ep *usrv.Endpoint) error {
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}

		// Wrap original method
		originalHandler := ep.Handler
		ep.Handler = usrv.HandlerFunc(func(ctx context.Context, responseWriter usrv.ResponseWriter, request *usrv.Message) {
			var traceId string

			// Check if the request contains a trace id. If no trace is
			// available allocate a new traceId and inject it in the
			// request context that gets passed to the handler
			trace := request.Headers.Get(CtxTraceId)
			if trace == nil {
				traceId = uuid.New()
				ctx = context.WithValue(ctx, CtxTraceId, traceId)
			} else {
				traceId = trace.(string)
			}

			// Inject trace into outgoing message
			responseWriter.Header().Set(CtxTraceId, traceId)

			// Trace incoming request. Use a select statement to ensure write is non-blocking.
			traceEntry := tracePkg.Record{
				Timestamp:     time.Now(),
				TraceId:       traceId,
				CorrelationId: request.CorrelationId,
				Type:          tracePkg.Request,
				From:          request.From,
				To:            request.To,
				Host:          hostname,
			}
			select {
			case collector.TraceChan <- traceEntry:
			// trace successfully added to channel
			default:
				// channel is full, skip trace
			}

			// Trace response when the handler returns
			defer func(start time.Time) {

				var errMsg string

				errVal := responseWriter.Header().Get("error")
				if errVal != nil {
					errMsg = errVal.(string)
				}

				traceEntry := tracePkg.Record{
					Timestamp:     time.Now(),
					TraceId:       traceId,
					CorrelationId: request.CorrelationId,
					Type:          tracePkg.Response,
					From:          request.To, // when responding we switch From/To
					To:            request.From,
					Host:          hostname,
					Duration:      time.Since(start).Nanoseconds(),
					Error:         errMsg,
				}

				select {
				case collector.TraceChan <- traceEntry:
				// trace successfully added to channel
				default:
					// channel is full, skip trace
				}
			}(time.Now())

			// Invoke the original handler
			originalHandler.Serve(ctx, responseWriter, request)
		})

		return nil
	}
}
