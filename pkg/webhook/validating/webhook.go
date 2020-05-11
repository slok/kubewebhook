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
}

func (c *WebhookConfig) validate() error {
	errs := ""

	if c.Name == "" {
		errs = errs + "name can't be empty"
	}

	if errs != "" {
		return fmt.Errorf("invalid configuration: %s", errs)
	}

	return nil
}

// NewWebhook is a validating webhook and will return a webhook ready for a type of resource
// it will validate the received resources.
func NewWebhook(cfg WebhookConfig, validator Validator, ot opentracing.Tracer, recorder metrics.Recorder, logger log.Logger) (webhook.Webhook, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	if logger == nil {
		logger = log.Dummy
	}

	if recorder == nil {
		logger.Warningf("no metrics recorder active")
		recorder = metrics.Dummy
	}

	if ot == nil {
		logger.Warningf("no tracer active")
		ot = &opentracing.NoopTracer{}
	}

	// If we don't have the type of the object, use a dynamic object creator that will
	// infer the type.
	var oc helpers.ObjectCreator = helpers.DynamicObjectCreator
	if cfg.Obj != nil {
		oc = helpers.NewStaticObjectCreator(cfg.Obj)
	}

	// Create our webhook and wrap for instrumentation (metrics and tracing).
	return &instrumenting.Webhook{
		Webhook: &validateWebhook{
			objectCreator: oc,
			validator:     validator,
			cfg:           cfg,
			logger:        logger,
		},
		ReviewKind:      metrics.ValidatingReviewKind,
		WebhookName:     cfg.Name,
		MetricsRecorder: recorder,
		Tracer:          ot,
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

	// Create a new object from the raw type.
	runtimeObj, err := w.objectCreator.NewObject(ar.Request.Object.Raw)
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
