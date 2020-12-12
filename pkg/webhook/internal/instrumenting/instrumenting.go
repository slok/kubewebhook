package instrumenting

/*
import (
	"context"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	opentracingext "github.com/opentracing/opentracing-go/ext"

	"github.com/slok/kubewebhook/pkg/model"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/internal/helpers"
)

// Webhook is a webhook wrapper that instruments the webhook with metrics and tracing.
// To the end user this webhook is transparent but internally instrumenting as a webhook
// wrapper we split responsibility.
type Webhook struct {
	Webhook         webhook.Webhook
	WebhookName     string
	ReviewKind      metrics.ReviewKind
	MetricsRecorder metrics.Recorder
	Tracer          opentracing.Tracer
}

// Review will review using the webhook wrapping it with instrumentation.
func (w *Webhook) Review(ctx context.Context, ar model.AdmissionReview) (model.AdmissionResponse, error) {
	// Initialize metrics.
	w.incAdmissionReviewMetric(ar, false)
	start := time.Now()
	defer w.observeAdmissionReviewDuration(ar, start)

	// Create the span, add to the context and defer the finish of the span.
	span := w.createReviewSpan(ctx, ar)
	ctx = opentracing.ContextWithSpan(ctx, span)
	defer span.Finish()

	// Call the review process.
	span.LogKV("event", "start_review")
	resp, err := w.Webhook.Review(ctx, ar)
	if err != nil {
		w.incAdmissionReviewMetric(ar, true)
		opentracingext.Error.Set(span, true)
		span.LogKV(
			"event", "error",
			// TODO(slok)
			//"message", resp.Result.Message,
		)
		return resp, err
	}

	// If its a validating response then increase our metric counter
	// TODO.
	// if !resp.Mutation {
	// 	w.incValidationReviewResultMetric(ar, resp.Allowed)
	// }

	//var msg, status string
	//if resp.Result != nil {
	//	// TODO(slok)
	//	msg = resp.Result.Message
	//	status = resp.Result.Status
	//}
	// TODO.
	// span.LogKV(
	// 	"event", "end_review",
	// 	"allowed", resp.Allowed,
	// 	"message", msg,
	// 	"patch", string(resp.JSONPatchPatch),
	// 	"status", status,
	// )

	return resp, nil
}

func (w *Webhook) incAdmissionReviewMetric(ar model.AdmissionReview, err bool) {
	if err {
		w.MetricsRecorder.IncAdmissionReviewError(
			w.WebhookName,
			ar.Namespace,
			helpers.GroupVersionResourceToString(*ar.RequestGVR),
			ar.Operation,
			w.ReviewKind)
	} else {
		w.MetricsRecorder.IncAdmissionReview(
			w.WebhookName,
			ar.Namespace,
			helpers.GroupVersionResourceToString(*ar.RequestGVR),
			ar.Operation,
			w.ReviewKind)
	}
}

func (w *Webhook) observeAdmissionReviewDuration(ar model.AdmissionReview, start time.Time) {
	w.MetricsRecorder.ObserveAdmissionReviewDuration(
		w.WebhookName,
		ar.Namespace,
		helpers.GroupVersionResourceToString(*ar.RequestGVR),
		ar.Operation,
		w.ReviewKind,
		start)
}

func (w *Webhook) incValidationReviewResultMetric(ar model.AdmissionReview, allowed bool) {
	w.MetricsRecorder.IncValidationReviewResult(
		w.WebhookName,
		ar.Namespace,
		helpers.GroupVersionResourceToString(*ar.RequestGVR),
		ar.Operation,
		allowed,
	)
}

func (w *Webhook) createReviewSpan(ctx context.Context, ar model.AdmissionReview) opentracing.Span {
	var spanOpts []opentracing.StartSpanOption

	// Check if we receive a previous span or we are the root span.
	if pSpan := opentracing.SpanFromContext(ctx); pSpan != nil {
		spanOpts = append(spanOpts, opentracing.ChildOf(pSpan.Context()))
	}

	// Create a new span.
	span := w.Tracer.StartSpan("review", spanOpts...)

	// Set span data.
	opentracingext.Component.Set(span, "kubewebhook")
	opentracingext.SpanKindRPCServer.Set(span)
	span.SetTag("kubewebhook.webhook.kind", w.ReviewKind)
	span.SetTag("kubewebhook.webhook.name", w.WebhookName)

	span.SetTag("kubernetes.review.uid", ar.ID)
	span.SetTag("kubernetes.review.namespace", ar.Namespace)
	span.SetTag("kubernetes.review.name", ar.Name)
	span.SetTag("kubernetes.review.objectKind", helpers.GroupVersionResourceToString(*ar.RequestGVR))

	return span
}
*/
