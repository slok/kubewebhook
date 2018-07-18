package mutating

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/appscode/jsonpatch"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/internal/helpers"
)

// WebhookConfig is the Mutating webhook configuration.
type WebhookConfig struct {
	Name string
	Obj  metav1.Object
}

func (c *WebhookConfig) validate() error {
	errs := ""

	if c.Name == "" {
		errs = errs + "name can't be empty"
	}

	if c.Obj == nil {
		errs = errs + "; obj can't be nil"
	}

	if errs != "" {
		return fmt.Errorf("invalid configuration: %s", errs)
	}

	return nil
}

type staticWebhook struct {
	objType      reflect.Type
	deserializer runtime.Decoder
	mutator      Mutator
	mRecorder    metrics.Recorder
	cfg          WebhookConfig
	logger       log.Logger
}

// NewWebhook is a mutating webhook and will return a webhook ready for a type of resource.
// It will mutate the received resources.
// This webhook will always allow the admission of the resource, only will deny in case of error.
func NewWebhook(cfg WebhookConfig, mutator Mutator, recorder metrics.Recorder, logger log.Logger) (webhook.Webhook, error) {
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

	// Create a custom deserializer for the received admission review request.
	runtimeScheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(runtimeScheme)

	return &staticWebhook{
		objType:      helpers.GetK8sObjType(cfg.Obj),
		deserializer: codecs.UniversalDeserializer(),
		mutator:      mutator,
		cfg:          cfg,
		mRecorder:    recorder,
		logger:       logger,
	}, nil
}

func (w *staticWebhook) Review(ctx context.Context, ar *admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	w.incAdmissionReviewMetric(ar, false)
	start := time.Now()
	defer w.observeAdmissionReviewDuration(ar, start)

	auid := ar.Request.UID

	w.logger.Debugf("reviewing request %s, named: %s/%s", auid, ar.Request.Namespace, ar.Request.Name)
	obj := helpers.NewK8sObj(w.objType)
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		err := fmt.Errorf("could not type assert metav1.Object to runtime.Object")
		return w.toAdmissionErrorResponse(ar, err)
	}

	// Get the object.
	_, _, err := w.deserializer.Decode(ar.Request.Object.Raw, nil, runtimeObj)
	if err != nil {
		err = fmt.Errorf("error deseralizing request raw object: %s", err)
		return w.toAdmissionErrorResponse(ar, err)
	}

	// Copy the object to have the original and be able to get the patch.
	objCopy := runtimeObj.DeepCopyObject()
	mutatingObj, ok := objCopy.(metav1.Object)
	if !ok {
		err := fmt.Errorf("impossible to type assert the deep copy to metav1.Object")
		return w.toAdmissionErrorResponse(ar, err)
	}

	return w.mutatingAdmissionReview(ctx, ar, obj, mutatingObj)

}

func (w *staticWebhook) mutatingAdmissionReview(ctx context.Context, ar *admissionv1beta1.AdmissionReview, obj, copyObj metav1.Object) *admissionv1beta1.AdmissionResponse {
	auid := ar.Request.UID

	// Mutate the object.
	_, err := w.mutator.Mutate(ctx, copyObj)
	if err != nil {
		return w.toAdmissionErrorResponse(ar, err)
	}

	// Get the diff patch of the original and mutated object.
	origJSON, err := json.Marshal(obj)
	if err != nil {
		return w.toAdmissionErrorResponse(ar, err)

	}
	mutatedJSON, err := json.Marshal(copyObj)
	if err != nil {
		return w.toAdmissionErrorResponse(ar, err)
	}

	patch, err := jsonpatch.CreatePatch(origJSON, mutatedJSON)
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

func (w *staticWebhook) toAdmissionErrorResponse(ar *admissionv1beta1.AdmissionReview, err error) *admissionv1beta1.AdmissionResponse {
	w.incAdmissionReviewMetric(ar, true)
	return helpers.ToAdmissionErrorResponse(ar.Request.UID, err, w.logger)
}

func (w *staticWebhook) incAdmissionReviewMetric(ar *admissionv1beta1.AdmissionReview, err bool) {
	if err {
		w.mRecorder.IncAdmissionReviewError(
			w.cfg.Name,
			ar.Request.Namespace,
			helpers.GroupVersionResourceToString(ar.Request.Resource),
			ar.Request.Operation,
			metrics.MutatingReviewKind)
	} else {
		w.mRecorder.IncAdmissionReview(
			w.cfg.Name,
			ar.Request.Namespace,
			helpers.GroupVersionResourceToString(ar.Request.Resource),
			ar.Request.Operation,
			metrics.MutatingReviewKind)
	}
}

func (w *staticWebhook) observeAdmissionReviewDuration(ar *admissionv1beta1.AdmissionReview, start time.Time) {
	w.mRecorder.ObserveAdmissionReviewDuration(
		w.cfg.Name,
		ar.Request.Namespace,
		helpers.GroupVersionResourceToString(ar.Request.Resource),
		ar.Request.Operation,
		metrics.MutatingReviewKind,
		start)
}

// jsonPatchType is the type for Kubernetes responses type.
var jsonPatchType = func() *admissionv1beta1.PatchType {
	pt := admissionv1beta1.PatchTypeJSONPatch
	return &pt
}()
