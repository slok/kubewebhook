package validating

import (
	"context"
	"fmt"

	opentracing "github.com/opentracing/opentracing-go"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/internal/helpers"
	"github.com/slok/kubewebhook/pkg/webhook/internal/instrumenting"
)

// WebhookConfig is the Validating webhook configuration.
type WebhookConfig struct {
	// Name is the name of the webhook.
	Name string
	// Object is the object of the webhook, to use multiple types on the same webhook or
	// type inference, don't set this field (will be `nil`).
	Obj metav1.Object
	// Validator is the webhook validator.
	Validator Validator
	// Tracer is the open tracing Tracer.
	Tracer opentracing.Tracer
	// MetricsRecorder is the metrics recorder.
	MetricsRecorder metrics.Recorder
	// Logger is the app logger.
	Logger log.Logger
}

func (c *WebhookConfig) defaults() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	if c.Validator == nil {
		return fmt.Errorf("validator is required")
	}

	if c.Logger == nil {
		c.Logger = log.Dummy
	}

	if c.MetricsRecorder == nil {
		c.MetricsRecorder = metrics.Dummy
	}

	if c.Tracer == nil {
		c.Tracer = &opentracing.NoopTracer{}
	}

	return nil
}

// NewWebhook is a validating webhook and will return a webhook ready for a type of resource
// it will validate the received resources.
func NewWebhook(cfg WebhookConfig) (webhook.Webhook, error) {
	if err := cfg.defaults(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// If we don't have the type of the object create a dynamic object creator that will
	// infer the type.
	var oc helpers.ObjectCreator
	if cfg.Obj != nil {
		oc = helpers.NewStaticObjectCreator(cfg.Obj)
	} else {
		oc = helpers.NewDynamicObjectCreator()
	}

	// Create our webhook and wrap for instrumentation (metrics and tracing).
	return &instrumenting.Webhook{
		Webhook: &validateWebhook{
			objectCreator: oc,
			validator:     cfg.Validator,
			cfg:           cfg,
			logger:        cfg.Logger,
		},
		ReviewKind:      metrics.ValidatingReviewKind,
		WebhookName:     cfg.Name,
		MetricsRecorder: cfg.MetricsRecorder,
		Tracer:          cfg.Tracer,
	}, nil
}

type validateWebhook struct {
	objectCreator helpers.ObjectCreator
	validator     Validator
	cfg           WebhookConfig
	logger        log.Logger
}

func (w validateWebhook) Review(ctx context.Context, ar *admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	w.logger.Debugf("reviewing request %s, named: %s/%s", ar.Request.UID, ar.Request.Namespace, ar.Request.Name)

	// Delete operations don't have body because should be gone on the deletion, instead they have the body
	// of the object we want to delete as an old object.
	raw := ar.Request.Object.Raw
	if ar.Request.Operation == admissionv1beta1.Delete {
		raw = ar.Request.OldObject.Raw
	}

	// Create a new object from the raw type.
	runtimeObj, err := w.objectCreator.NewObject(raw)
	if err != nil {
		return w.toAdmissionErrorResponse(ar, err)
	}

	validatingObj, ok := runtimeObj.(metav1.Object)
	// Get the object.
	if !ok {
		err := fmt.Errorf("impossible to type assert the deep copy to metav1.Object")
		return w.toAdmissionErrorResponse(ar, err)
	}

	_, res, err := w.validator.Validate(ctx, validatingObj)
	if err != nil {
		return w.toAdmissionErrorResponse(ar, err)
	}

	var status string
	if res.Valid {
		status = metav1.StatusSuccess
	}

	// Forge response.
	return &admissionv1beta1.AdmissionResponse{
		UID:     ar.Request.UID,
		Allowed: res.Valid,
		Result: &metav1.Status{
			Status:  status,
			Message: res.Message,
		},
	}
}

func (w validateWebhook) toAdmissionErrorResponse(ar *admissionv1beta1.AdmissionReview, err error) *admissionv1beta1.AdmissionResponse {
	return helpers.ToAdmissionErrorResponse(ar.Request.UID, err, w.logger)
}
