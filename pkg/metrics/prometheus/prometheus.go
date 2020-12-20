package prometheus

import (
	"context"
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/slok/kubewebhook/v2/pkg/webhook"
)

const (
	prefix = "kubewebhook"
)

// RecorderConfig is the configuration of the recorder.
type RecorderConfig struct {
	Registry        prometheus.Registerer
	ReviewOpBuckets []float64
}

func (c *RecorderConfig) defaults() error {
	if c.Registry == nil {
		c.Registry = prometheus.DefaultRegisterer
	}

	if c.ReviewOpBuckets == nil {
		c.ReviewOpBuckets = prometheus.DefBuckets
	}

	return nil
}

// Recorder knows how to measure the metrics of the library using Prometheus
// as the backend for the measurements.
type Recorder struct {
	webhookValReviewDuration *prometheus.HistogramVec
	webhookMutReviewDuration *prometheus.HistogramVec
	webhookReviewWarnings    *prometheus.CounterVec
}

// NewRecorder returns a new Prometheus metrics recorder.
func NewRecorder(config RecorderConfig) (*Recorder, error) {
	err := config.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	r := &Recorder{
		webhookValReviewDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prefix,
			Subsystem: "validating_webhook",
			Name:      "review_duration_seconds",
			Help:      "The duration of the admission review handled by a validating webhook.",
			Buckets:   config.ReviewOpBuckets,
		}, []string{"webhook_id", "webhook_version", "resource_namespace", "resource_kind", "operation", "dry_run", "success", "allowed"}),

		webhookMutReviewDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prefix,
			Subsystem: "mutating_webhook",
			Name:      "review_duration_seconds",
			Help:      "The duration of the admission review handled by a mutating webhook.",
			Buckets:   config.ReviewOpBuckets,
		}, []string{"webhook_id", "webhook_version", "resource_namespace", "resource_kind", "operation", "dry_run", "success", "mutated"}),

		webhookReviewWarnings: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prefix,
			Subsystem: "webhook",
			Name:      "review_warnings_total",
			Help:      "The total number warnings the webhooks are returning on the review process.",
		}, []string{"webhook_id", "webhook_version", "resource_namespace", "resource_kind", "operation", "dry_run", "success"}),
	}

	// Register our metrics on the received recorder.
	config.Registry.MustRegister(
		r.webhookValReviewDuration,
		r.webhookMutReviewDuration,
		r.webhookReviewWarnings,
	)

	return r, nil
}

var _ webhook.MetricsRecorder = Recorder{}

// MeasureValidatingWebhookReviewOp measures a validating webhook review operation on Prometheus.
func (r Recorder) MeasureValidatingWebhookReviewOp(_ context.Context, data webhook.MeasureValidatingOpData) {
	// Measure Operation.
	r.webhookValReviewDuration.With(prometheus.Labels{
		"webhook_id":         data.WebhookID,
		"webhook_version":    data.AdmissionReviewVersion,
		"resource_namespace": data.ResourceNamespace,
		"resource_kind":      data.ResourceKind,
		"operation":          data.Operation,
		"dry_run":            strconv.FormatBool(data.DryRun),
		"success":            strconv.FormatBool(data.Success),
		"allowed":            strconv.FormatBool(data.Allowed),
	}).Observe(data.Duration.Seconds())

	// Measure warnings.
	r.webhookReviewWarnings.With(prometheus.Labels{
		"webhook_id":         data.WebhookID,
		"webhook_version":    data.AdmissionReviewVersion,
		"resource_namespace": data.ResourceNamespace,
		"resource_kind":      data.ResourceKind,
		"operation":          data.Operation,
		"dry_run":            strconv.FormatBool(data.DryRun),
		"success":            strconv.FormatBool(data.Success),
	}).Add(float64(data.WarningsNumber))
}

// MeasureMutatingWebhookReviewOp measures a mutating webhook review operation on Prometheus.
func (r Recorder) MeasureMutatingWebhookReviewOp(_ context.Context, data webhook.MeasureMutatingOpData) {
	// Measure operation.
	r.webhookMutReviewDuration.With(prometheus.Labels{
		"webhook_id":         data.WebhookID,
		"webhook_version":    data.AdmissionReviewVersion,
		"resource_namespace": data.ResourceNamespace,
		"resource_kind":      data.ResourceKind,
		"operation":          data.Operation,
		"dry_run":            strconv.FormatBool(data.DryRun),
		"success":            strconv.FormatBool(data.Success),
		"mutated":            strconv.FormatBool(data.Mutated),
	}).Observe(data.Duration.Seconds())

	// Measure warnings.
	r.webhookReviewWarnings.With(prometheus.Labels{
		"webhook_id":         data.WebhookID,
		"webhook_version":    data.AdmissionReviewVersion,
		"resource_namespace": data.ResourceNamespace,
		"resource_kind":      data.ResourceKind,
		"operation":          data.Operation,
		"dry_run":            strconv.FormatBool(data.DryRun),
		"success":            strconv.FormatBool(data.Success),
	}).Add(float64(data.WarningsNumber))
}
