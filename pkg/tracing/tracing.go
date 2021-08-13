package tracing

import (
	"context"
	"net/http"
)

// Tracer is the interface a tracer for the application should implement.
type Tracer interface {
	// WithValues returns a new tracer that will add the values to all the
	// traces created by the returned tracer.
	WithValues(values map[string]interface{}) Tracer
	// NewTrace returns a context with a trace created in it.
	NewTrace(ctx context.Context, name string) context.Context
	// EndTrace ends the trace that is currently on the context.
	// If there are no traces on the context it will be a noop.
	EndTrace(ctx context.Context, err error)
	// TraceHTTPHandler returns a new http.Handler wrapped with the required
	// things to trace the original HTTP handler execution.
	TraceHTTPHandler(name string, h http.Handler) http.Handler
	// TraceHTTPClient returns a new http.Client based on the received one
	// this new Client will trace all the HTTP requests executed with the
	// client.
	//
	// Note: To trace correctly from parent trace/spans, the requests used by the client
	// should have the context set on the request.
	TraceHTTPClient(name string, c *http.Client) *http.Client
	// TraceFunc is a helper that executes a function and trace its execution, the execution
	// can return values and errors that will be used for the trace information.
	TraceFunc(ctx context.Context, name string, f func(ctx context.Context) (values map[string]interface{}, err error))
	// TraceID returns the current trace ID. This is useful to measure/record somewhere and point
	// the current trace (e.g with a logger).
	TraceID(ctx context.Context) string
	// AddTraceValues adds values to the current context trace.
	// If there are not traces on the context is a noop.
	AddTraceValues(ctx context.Context, values map[string]interface{})
	// AddTraceEvent adds an event on the current context trace.
	// If there are not traces on the context is a noop.
	AddTraceEvent(ctx context.Context, event string, values map[string]interface{})
}

// Noop tracer doesn't trace anything.
const Noop = noop(0)

type noop int

func (n noop) WithValues(values map[string]interface{}) Tracer           { return n }
func (n noop) TraceID(ctx context.Context) string                        { return "" }
func (n noop) NewTrace(ctx context.Context, name string) context.Context { return ctx }
func (n noop) EndTrace(ctx context.Context, err error)                   {}
func (n noop) TraceHTTPHandler(name string, h http.Handler) http.Handler { return h }
func (n noop) TraceHTTPClient(name string, c *http.Client) *http.Client  { return c }
func (n noop) TraceFunc(ctx context.Context, name string, f func(ctx context.Context) (values map[string]interface{}, err error)) {
	_, _ = f(ctx)
}
func (n noop) AddTraceValues(ctx context.Context, values map[string]interface{})              {}
func (n noop) AddTraceEvent(ctx context.Context, event string, values map[string]interface{}) {}
