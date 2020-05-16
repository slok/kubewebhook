package mutating

import (
	"context"
	"encoding/json"
	"fmt"

	opentracing "github.com/opentracing/opentracing-go"
	"gomodules.xyz/jsonpatch/v3"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/internal/helpers"
	"github.com/slok/kubewebhook/pkg/webhook/internal/instrumenting"
)

// WebhookConfig is the Mutating webhook configuration.
type WebhookConfig struct {
	// Name is the name of the webhook.
	Name string
	// Object is the object of the webhook, to use multiple types on the same webhook or
	// type inference, don't set this field (will be `nil`).
	Obj metav1.Object
}

func (c WebhookConfig) validate() error {
	errs := ""

	if c.Name == "" {
		errs = errs + "name can't be empty"
	}

	if errs != "" {
		return fmt.Errorf("invalid configuration: %s", errs)
	}

	return nil
}

type mutationWebhook struct {
	objectCreator helpers.ObjectCreator
	mutator       Mutator
	cfg           WebhookConfig
	logger        log.Logger
}

// NewWebhook is a mutating webhook and will return a webhook ready for a type of resource.
// It will mutate the received resources.
// This webhook will always allow the admission of the resource, only will deny in case of error.
func NewWebhook(cfg WebhookConfig, mutator Mutator, ot opentracing.Tracer, recorder metrics.Recorder, logger log.Logger) (webhook.Webhook, error) {
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
		Webhook: &mutationWebhook{
			objectCreator: oc,
			mutator:       mutator,
			cfg:           cfg,
			logger:        logger,
		},
		ReviewKind:      metrics.MutatingReviewKind,
		WebhookName:     cfg.Name,
		MetricsRecorder: recorder,
		Tracer:          ot,
	}, nil
}

func (w mutationWebhook) Review(ctx context.Context, ar *admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	auid := ar.Request.UID

	w.logger.Debugf("reviewing request %s, named: %s/%s", auid, ar.Request.Namespace, ar.Request.Name)

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

	mutatingObj, ok := runtimeObj.(metav1.Object)
	if !ok {
		err := fmt.Errorf("impossible to type assert the deep copy to metav1.Object")
		return w.toAdmissionErrorResponse(ar, err)
	}

	return w.mutatingAdmissionReview(ctx, ar, raw, mutatingObj)

}

func (w mutationWebhook) mutatingAdmissionReview(ctx context.Context, ar *admissionv1beta1.AdmissionReview, rawObj []byte, obj metav1.Object) *admissionv1beta1.AdmissionResponse {
	auid := ar.Request.UID

	// Mutate the object.
	_, err := w.mutator.Mutate(ctx, obj)
	if err != nil {
		return w.toAdmissionErrorResponse(ar, err)
	}

	mutatedJSON, err := json.Marshal(obj)
	if err != nil {
		return w.toAdmissionErrorResponse(ar, err)
	}

	patch, err := jsonpatch.CreatePatch(rawObj, mutatedJSON)
	if err != nil {
		return w.toAdmissionErrorResponse(ar, err)
	}

	marshalledPatch, err := json.Marshal(patch)
	if err != nil {
		return w.toAdmissionErrorResponse(ar, err)
	}
	w.logger.Debugf("json patch for request %s: %s", auid, string(marshalledPatch))

	// Forge response.
	return &admissionv1beta1.AdmissionResponse{
		UID:       auid,
		Allowed:   true,
		Patch:     marshalledPatch,
		PatchType: jsonPatchType,
	}
}

func (w mutationWebhook) toAdmissionErrorResponse(ar *admissionv1beta1.AdmissionReview, err error) *admissionv1beta1.AdmissionResponse {
	return helpers.ToAdmissionErrorResponse(ar.Request.UID, err, w.logger)
}

// jsonPatchType is the type for Kubernetes responses type.
var jsonPatchType = func() *admissionv1beta1.PatchType {
	pt := admissionv1beta1.PatchTypeJSONPatch
	return &pt
}()
