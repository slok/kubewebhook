package otel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/slok/kubewebhook/v2/pkg/tracing"
)

const tracerName = "github.com/slok/kubewebhook"

var errNoSpanOnContext = errors.New("no span on context")

type tracer struct {
	otelTracerProvider oteltrace.TracerProvider
	otelTracer         oteltrace.Tracer
	otelPropagator     propagation.TextMapPropagator
	values             map[string]interface{}
}

// NewTracer returns a new Open telemetry tracer.
func NewTracer(otelTracerProvider oteltrace.TracerProvider, otelPropagator propagation.TextMapPropagator) tracing.Tracer {
	return tracer{
		otelTracerProvider: otelTracerProvider,
		otelTracer:         otelTracerProvider.Tracer(tracerName),
		otelPropagator:     otelPropagator,
	}
}

func (t tracer) WithValues(values map[string]interface{}) tracing.Tracer {
	return tracer{
		otelTracerProvider: t.otelTracerProvider,
		otelTracer:         t.otelTracer,
		otelPropagator:     t.otelPropagator,
		values:             values,
	}
}

func (t tracer) TraceID(ctx context.Context) string {
	span, err := t.spanFromContext(ctx)
	if err != nil {
		return ""
	}

	return span.SpanContext().TraceID().String()
}

func (t tracer) spanFromContext(ctx context.Context) (oteltrace.Span, error) {
	otelSpan := oteltrace.SpanFromContext(ctx)

	// Is there any span on the context?.
	// Check if noop: https://github.com/open-telemetry/opentelemetry-go/blob/39fe8092ed0156b6cbb8225589a81b86124fa491/trace/noop.go#L57
	if !otelSpan.IsRecording() {
		return nil, errNoSpanOnContext
	}

	return otelSpan, nil
}

func (t tracer) NewTrace(ctx context.Context, name string) context.Context {
	ctx, _ = t.newOtelSpan(ctx, name)
	return ctx
}

func (t tracer) EndTrace(ctx context.Context, e error) {
	span, err := t.spanFromContext(ctx)
	if err != nil {
		return
	}

	if e != nil {
		span.RecordError(e)
		span.SetStatus(codes.Error, e.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.End()
}

func (t tracer) AddTraceValues(ctx context.Context, values map[string]interface{}) {
	span, err := t.spanFromContext(ctx)
	if err != nil {
		return
	}

	span.SetAttributes(
		t.mapValuesToOtelAttributes(values)...,
	)
}

func (t tracer) TraceHTTPHandler(name string, h http.Handler) http.Handler {
	return otelhttp.NewHandler(h, name,
		otelhttp.WithSpanOptions(oteltrace.WithAttributes(
			t.mapValuesToOtelAttributes(t.values)...,
		)),
		otelhttp.WithTracerProvider(t.otelTracerProvider),
		otelhttp.WithPropagators(t.otelPropagator))
}

func (t tracer) TraceHTTPClient(name string, c *http.Client) *http.Client {
	opts := []otelhttp.Option{
		otelhttp.WithSpanOptions(oteltrace.WithAttributes(
			t.mapValuesToOtelAttributes(t.values)...,
		)),
		otelhttp.WithTracerProvider(t.otelTracerProvider),
		otelhttp.WithPropagators(t.otelPropagator),
	}

	// Set custom formatter.
	if name != "" {
		opts = append(opts, otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return fmt.Sprintf("%s: HTTP %s", name, r.Method)
		}))
	}

	return &http.Client{
		CheckRedirect: c.CheckRedirect,
		Jar:           c.Jar,
		Timeout:       c.Timeout,
		Transport:     otelhttp.NewTransport(c.Transport, opts...),
	}
}

func (t tracer) TraceFunc(ctx context.Context, name string, f func(ctx context.Context) (values map[string]interface{}, err error)) {
	ctx, span := t.newOtelSpan(ctx, name)

	values, err := f(ctx)

	span.SetAttributes(
		t.mapValuesToOtelAttributes(values)...,
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.End()
}

func (t tracer) AddTraceEvent(ctx context.Context, event string, values map[string]interface{}) {
	span, err := t.spanFromContext(ctx)
	if err != nil {
		return
	}

	attrs := t.mapValuesToOtelAttributes(values)
	span.AddEvent(event, oteltrace.WithAttributes(attrs...))
}

func (t tracer) newOtelSpan(ctx context.Context, name string) (context.Context, oteltrace.Span) {
	return t.otelTracer.Start(ctx, name, oteltrace.WithAttributes(
		t.mapValuesToOtelAttributes(t.values)...,
	))
}

func (t tracer) mapValuesToOtelAttributes(values map[string]interface{}) []attribute.KeyValue {
	kvs := make([]attribute.KeyValue, 0, len(values))
	for k, v := range values {
		kvs = append(kvs, any(k, v))
	}

	return kvs
}

func any(k string, v interface{}) attribute.KeyValue {
	if v == nil {
		return attribute.String(k, "<nil>")
	}

	switch typed := v.(type) {
	case bool:
		return attribute.Bool(k, typed)
	case []bool:
		return attribute.BoolSlice(k, typed)
	case int:
		return attribute.Int(k, typed)
	case []int:
		return attribute.IntSlice(k, typed)
	case int8:
		return attribute.Int(k, int(typed))
	case []int8:
		ls := make([]int, 0, len(typed))
		for _, i := range typed {
			ls = append(ls, int(i))
		}
		return attribute.IntSlice(k, ls)
	case int16:
		return attribute.Int(k, int(typed))
	case []int16:
		ls := make([]int, 0, len(typed))
		for _, i := range typed {
			ls = append(ls, int(i))
		}
		return attribute.IntSlice(k, ls)
	case int32:
		return attribute.Int64(k, int64(typed))
	case []int32:
		ls := make([]int64, 0, len(typed))
		for _, i := range typed {
			ls = append(ls, int64(i))
		}
		return attribute.Int64Slice(k, ls)
	case int64:
		return attribute.Int64(k, typed)
	case []int64:
		return attribute.Int64Slice(k, typed)
	case float64:
		return attribute.Float64(k, typed)
	case []float64:
		return attribute.Float64Slice(k, typed)
	case string:
		return attribute.String(k, typed)
	case []string:
		return attribute.StringSlice(k, typed)
	}

	if stringer, ok := v.(fmt.Stringer); ok {
		return attribute.String(k, stringer.String())
	}
	if b, err := json.Marshal(v); b != nil && err == nil {
		return attribute.String(k, string(b))
	}
	return attribute.String(k, fmt.Sprintf("%v", v))
}
