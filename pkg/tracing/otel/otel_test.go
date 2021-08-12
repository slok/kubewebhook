package otel_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/slok/kubewebhook/v2/pkg/tracing"
	"github.com/slok/kubewebhook/v2/pkg/tracing/otel"
)

func TestTracer(t *testing.T) {
	tests := map[string]struct {
		exec   func(ctx context.Context, tracer tracing.Tracer)
		expect func(t *testing.T, spans []sdktrace.ReadOnlySpan)
	}{
		"If no trace on context, ending a trace shouldn't do anything.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				tracer.EndTrace(ctx, nil)
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				assertSpans(t, []expSpan{}, spans, nil)
			},
		},

		"If no trace on context, adding values shouldn't do anything.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				tracer.AddTraceValues(ctx, map[string]interface{}{"k1": "v1"})
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				assertSpans(t, []expSpan{}, spans, nil)
			},
		},

		"If no trace on context, adding events shouldn't do anything.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				tracer.AddTraceEvent(ctx, "ev1", map[string]interface{}{"k1": "v1"})
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				assertSpans(t, []expSpan{}, spans, nil)
			},
		},

		"If no trace on context, getting the trace ID shouldn't do anything.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				tracer.TraceID(ctx)
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				assertSpans(t, []expSpan{}, spans, nil)
			},
		},

		"An span without anything should trace correctly.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				ctx = tracer.NewTrace(ctx, "test-span1")
				tracer.EndTrace(ctx, nil)
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				exp := []expSpan{
					{name: "test-span1", kind: trace.SpanKindInternal, StatusCode: codes.Ok},
				}
				assertSpans(t, exp, spans, nil)
			},
		},

		"An span ended with error should trace as an error.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				ctx = tracer.NewTrace(ctx, "test-span1")
				tracer.EndTrace(ctx, fmt.Errorf("something"))
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				exp := []expSpan{
					{name: "test-span1", kind: trace.SpanKindInternal, StatusCode: codes.Error, StatusDesc: "something", events: []expEvent{
						{message: "exception", attrs: map[string]interface{}{"exception.message": "something", "exception.type": "*errors.errorString"}}},
					},
				}
				assertSpans(t, exp, spans, nil)
			},
		},

		"An span with tracer values should trace correctly.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				tracer = tracer.WithValues(map[string]interface{}{"source_attr": "tracer"})
				ctx = tracer.NewTrace(ctx, "test-span1")

				tracer.EndTrace(ctx, nil)
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				exp := []expSpan{
					{name: "test-span1", kind: trace.SpanKindInternal, StatusCode: codes.Ok, attrs: map[string]interface{}{
						"source_attr": "tracer",
					}},
				}
				assertSpans(t, exp, spans, nil)
			},
		},

		"Adding values to a span should trace correctly with the attributes.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				tracer = tracer.WithValues(map[string]interface{}{"source_attr": "tracer"})
				ctx = tracer.NewTrace(ctx, "test-span1")
				tracer.AddTraceValues(ctx, map[string]interface{}{"this_is": "a_test"})
				tracer.EndTrace(ctx, nil)
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				exp := []expSpan{
					{name: "test-span1", kind: trace.SpanKindInternal, StatusCode: codes.Ok, attrs: map[string]interface{}{
						"source_attr": "tracer",
						"this_is":     "a_test",
					}},
				}
				assertSpans(t, exp, spans, nil)
			},
		},

		"Span hierarchy should be respected.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				ctx1 := tracer.NewTrace(ctx, "test-span1")
				ctx11 := tracer.NewTrace(ctx1, "test-span1-1")
				ctx2 := tracer.NewTrace(ctx, "test-span2")
				ctx21 := tracer.NewTrace(ctx2, "test-span2-1")
				ctx111 := tracer.NewTrace(ctx11, "test-span1-1-1")
				tracer.EndTrace(ctx111, nil)
				tracer.EndTrace(ctx21, nil)
				tracer.EndTrace(ctx2, nil)
				tracer.EndTrace(ctx11, nil)
				tracer.EndTrace(ctx1, nil)
				tracer.EndTrace(ctx, nil)
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				exp := []expSpan{
					{name: "test-span1-1-1", kind: trace.SpanKindInternal, StatusCode: codes.Ok},
					{name: "test-span2-1", kind: trace.SpanKindInternal, StatusCode: codes.Ok},
					{name: "test-span2", kind: trace.SpanKindInternal, StatusCode: codes.Ok},
					{name: "test-span1-1", kind: trace.SpanKindInternal, StatusCode: codes.Ok},
					{name: "test-span1", kind: trace.SpanKindInternal, StatusCode: codes.Ok},
				}
				assertSpans(t, exp, spans, nil)

				// Check hierarchy.
				sp111 := spans[0]
				sp21 := spans[1]
				sp2 := spans[2]
				sp11 := spans[3]
				sp1 := spans[4]

				assertParent(t, sp11, sp111)
				assertParent(t, sp1, sp11)
				assertParent(t, sp2, sp21)
				assertParent(t, nil, sp2)
				assertParent(t, nil, sp1)
			},
		},

		"Tracing a func with values should trace the function correctly.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				tracer.TraceFunc(ctx, "test-span1", func(ctx context.Context) (map[string]interface{}, error) {
					return map[string]interface{}{"k1": "v1"}, nil
				})
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				exp := []expSpan{
					{name: "test-span1", kind: trace.SpanKindInternal, StatusCode: codes.Ok, attrs: map[string]interface{}{
						"k1": "v1",
					}},
				}
				assertSpans(t, exp, spans, nil)
			},
		},

		"Tracing a func with error should trace the function correctly.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				tracer.TraceFunc(ctx, "test-span1", func(ctx context.Context) (map[string]interface{}, error) {
					return nil, fmt.Errorf("something")
				})
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				exp := []expSpan{
					{name: "test-span1", kind: trace.SpanKindInternal, StatusCode: codes.Error, StatusDesc: "something", events: []expEvent{
						{message: "exception", attrs: map[string]interface{}{"exception.message": "something", "exception.type": "*errors.errorString"}}},
					},
				}
				assertSpans(t, exp, spans, nil)
			},
		},

		"Event on traces should record event.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				ctx = tracer.NewTrace(ctx, "test-span1")
				tracer.AddTraceEvent(ctx, "This is an event!", nil)
				tracer.EndTrace(ctx, nil)
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				exp := []expSpan{
					{name: "test-span1", kind: trace.SpanKindInternal, StatusCode: codes.Ok, events: []expEvent{
						{message: "This is an event!"},
					}},
				}
				assertSpans(t, exp, spans, nil)
			},
		},

		"Linked traces should have the same trace ID.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				ctx1 := tracer.NewTrace(ctx, "test-span1")
				defer tracer.EndTrace(ctx1, nil)
				traceID1 := tracer.TraceID(ctx1)
				ctx2 := tracer.NewTrace(ctx1, "test-span2")
				defer tracer.EndTrace(ctx2, nil)
				traceID2 := tracer.TraceID(ctx2)

				// Add the trace ID as values so we ca check it afterwards.
				tracer.AddTraceValues(ctx1, map[string]interface{}{"tid": traceID1})
				tracer.AddTraceValues(ctx2, map[string]interface{}{"tid": traceID2})
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				require.Len(t, spans, 2)

				// Check the trace ID from the values stored in the trace explicitly.
				span1TID := spans[0].Attributes()[0].Value.AsString()
				span2TID := spans[1].Attributes()[0].Value.AsString()

				assert.NotEmpty(t, span1TID)
				assert.NotEmpty(t, span2TID)
				assert.Equal(t, span1TID, span2TID)
			},
		},

		"Events with values on traces should record events.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				ctx = tracer.NewTrace(ctx, "test-span1")
				tracer.AddTraceEvent(ctx, "event 1", map[string]interface{}{"k1": "v1"})
				tracer.AddTraceEvent(ctx, "event 2", map[string]interface{}{"k2": "v2"})
				tracer.EndTrace(ctx, nil)
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				exp := []expSpan{
					{name: "test-span1", kind: trace.SpanKindInternal, StatusCode: codes.Ok, events: []expEvent{
						{message: "event 1", attrs: map[string]interface{}{"k1": "v1"}},
						{message: "event 2", attrs: map[string]interface{}{"k2": "v2"}},
					}},
				}
				assertSpans(t, exp, spans, nil)
			},
		},

		"HTTP handler should trace http requests correctly.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				h := tracer.TraceHTTPHandler("test-http-span1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusSeeOther)
					_, _ = w.Write([]byte("this is a test"))
				}))

				req := httptest.NewRequest(http.MethodGet, "/this/is/a/test", nil)
				var rec httptest.ResponseRecorder
				h.ServeHTTP(&rec, req)
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				exp := []expSpan{
					{name: "test-http-span1", kind: trace.SpanKindServer, StatusCode: codes.Unset, attrs: map[string]interface{}{
						"http.flavor":      "1.1",
						"http.host":        "example.com",
						"http.method":      "GET",
						"http.scheme":      "http",
						"http.server_name": "test-http-span1",
						"http.status_code": int64(303),
						"http.target":      "/this/is/a/test",
						"http.wrote_bytes": int64(14),
						"net.host.name":    "example.com",
						"net.peer.ip":      "192.0.2.1",
						"net.peer.port":    int64(1234),
						"net.transport":    "ip_tcp",
					}},
				}
				assertSpans(t, exp, spans, nil)
			},
		},

		"HTTP handler and client end to end should trace correctly on both sides.": {
			exec: func(ctx context.Context, tracer tracing.Tracer) {
				h := tracer.TraceHTTPHandler("test-http-server", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusAccepted)
					_, _ = w.Write([]byte("this is a test"))
				}))

				// Create a test server.
				server := httptest.NewServer(h)
				defer server.Close()

				// Create client.
				client := tracer.TraceHTTPClient("test-http-client", &http.Client{})

				// Make request with an internal trace before using the traced HTTP client.
				tracer.TraceFunc(ctx, "test-internal", func(ctx context.Context) (map[string]interface{}, error) {
					req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/this/is/a/test", nil)
					resp, err := client.Do(req)
					if err != nil {
						return nil, err
					}
					defer resp.Body.Close()

					return map[string]interface{}{"code": resp.StatusCode}, nil
				})
			},
			expect: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				// Check spans
				exp := []expSpan{
					{name: "test-internal", kind: trace.SpanKindInternal, StatusCode: codes.Ok, attrs: map[string]interface{}{
						"code": int64(202),
					}},
					{name: "test-http-server", kind: trace.SpanKindServer, StatusCode: codes.Unset, attrs: map[string]interface{}{
						"http.flavor":      "1.1",
						"http.host":        "", // Ignored.
						"http.method":      "GET",
						"http.scheme":      "http",
						"http.server_name": "test-http-server",
						"http.status_code": int64(202),
						"http.target":      "/this/is/a/test",
						"http.wrote_bytes": int64(14),
						"net.peer.ip":      "127.0.0.1",
						"net.peer.port":    0, // Ignored.
						"net.transport":    "ip_tcp",
						"net.host.ip":      "127.0.0.1",
						"net.host.port":    0, // Ignored.
						"http.user_agent":  "Go-http-client/1.1",
					}},
					{name: "test-http-client: HTTP GET", kind: trace.SpanKindClient, StatusCode: codes.Unset, attrs: map[string]interface{}{
						"http.method":      "GET",
						"http.url":         "", // Ignored.
						"http.scheme":      "http",
						"http.host":        "", // Ignored.
						"http.flavor":      "1.1",
						"http.status_code": int64(202),
					}},
				}
				assertSpans(t, exp, spans, []string{"http.host", "net.peer.port", "http.url", "net.host.port"})

				// Check hierarchy.
				var (
					internalSpan sdktrace.ReadOnlySpan
					clientSpan   sdktrace.ReadOnlySpan
					serverSpan   sdktrace.ReadOnlySpan
				)

				for _, span := range spans {
					switch span.Name() {
					case "test-http-server":
						serverSpan = span
					case "test-http-client: HTTP GET":
						clientSpan = span
					case "test-internal":
						internalSpan = span
					}
				}
				assertParent(t, clientSpan, serverSpan)
				assertParent(t, internalSpan, clientSpan)
				assertParent(t, nil, internalSpan)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Prepare.
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})

			tracer := otel.NewTracer(provider, propagator)

			// Execute.
			ctx := context.Background()
			test.exec(ctx, tracer)

			// Check.
			gotspans := sr.Ended()
			test.expect(t, gotspans)
		})
	}
}

type expSpan struct {
	name       string
	kind       trace.SpanKind
	attrs      map[string]interface{}
	events     []expEvent
	StatusCode codes.Code
	StatusDesc string
}

type expEvent struct {
	message string
	attrs   map[string]interface{}
}

func assertSpans(t *testing.T, exp []expSpan, spans []sdktrace.ReadOnlySpan, ignoreAttrs []string) {
	require.Len(t, spans, len(exp), "Invalid number of spans")

	// Index our expected spans by name (traces not always maintain order).
	expM := make(map[string]expSpan, len(exp))
	for _, v := range exp {
		expM[v.name] = v
	}

	// Check all got spans against the indexed expected ones.
	for _, got := range spans {
		if !assert.Contains(t, expM, got.Name()) {
			continue
		}
		assertSpan(t, expM[got.Name()], got, ignoreAttrs)
	}
}

func assertSpan(t *testing.T, exp expSpan, span sdktrace.ReadOnlySpan, ignoreAttrs []string) {
	// Check base information.
	assert.Equal(t, exp.name, span.Name(), "Invalid span name")
	assert.Equal(t, exp.kind, span.SpanKind(), "Invalid span kind")
	assert.Equal(t, exp.StatusCode, span.Status().Code, "Invalid status code")
	assert.Equal(t, exp.StatusDesc, span.Status().Description, "Invalid status description")

	// Check attributes.
	assertAttributes(t, exp.attrs, span.Attributes(), ignoreAttrs)

	// Check events.
	assert.Len(t, span.Events(), len(exp.events), "Invalid number of events")
	for i, expEvent := range exp.events {
		assert.Equal(t, expEvent.message, span.Events()[i].Name, "Invalid event message")
		assertAttributes(t, expEvent.attrs, span.Events()[i].Attributes, nil)
	}
}

func assertAttributes(t *testing.T, exp map[string]interface{}, attrs []attribute.KeyValue, ignoreAttrs []string) {
	assert.Len(t, attrs, len(exp), "Invalid number of attributes")

	ignores := make(map[string]struct{}, len(ignoreAttrs))
	for _, ignoreKey := range ignoreAttrs {
		ignores[ignoreKey] = struct{}{}
	}

	got := make(map[string]interface{}, len(attrs))

	for _, a := range attrs {
		got[string(a.Key)] = a.Value.AsInterface()
	}
	for k, v := range exp {
		if _, ok := ignores[k]; ok {
			continue
		}

		if !assert.Contains(t, got, k) {
			continue
		}
		assert.Equal(t, got[k], v)
	}
}

func assertParent(t *testing.T, parent, child sdktrace.ReadOnlySpan) {
	if parent == nil {
		assert.False(t, child.Parent().HasSpanID(), "Parent has span, root spans shouldn't")
		return
	}
	assert.Equal(t, parent.SpanContext().TraceID(), child.SpanContext().TraceID(), "Spans are not form the same trace")
	assert.Equal(t, parent.SpanContext().SpanID(), child.Parent().SpanID(), "Span is not parent")
}
