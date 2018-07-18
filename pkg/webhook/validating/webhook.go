package validating

import (
	"context"
	"fmt"
	"reflect"
	"time"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/internal/helpers"
)

// WebhookConfig is the Validating webhook configuration.
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
	validator    Validator
	mRecorder    metrics.Recorder
	cfg          WebhookConfig
	logger       log.Logger
}

// NewWebhook is a validating webhook and will return a webhook ready for a type of resource
// it will validate the received resources.
func NewWebhook(cfg WebhookConfig, validator Validator, recorder metrics.Recorder, logger log.Logger) (webhook.Webhook, error) {
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
		validator:    validator,
		mRecorder:    recorder,
		cfg:          cfg,
		logger:       logger,
	}, nil
}

func (w *staticWebhook) Review(ctx context.Context, ar *admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	w.incAdmissionReviewMetric(ar, false)
	start := time.Now()
	defer w.observeAdmissionReviewDuration(ar, start)

	w.logger.Debugf("reviewing request %s, named: %s/%s", ar.Request.UID, ar.Request.Namespace, ar.Request.Name)

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

	// Check validation on the object.
	_, res, err := w.validator.Validate(ctx, obj)
	if err != nil {
		return w.toAdmissionErrorResponse(ar, err)
	}

	// Forge response.
	return &admissionv1beta1.AdmissionResponse{
		UID:     ar.Request.UID,
		Allowed: res.Valid,
		Result: &metav1.Status{
			Status:  metav1.StatusSuccess,
			Message: res.Message,
		},
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
			metrics.ValidatingReviewKind)
	} else {
		w.mRecorder.IncAdmissionReview(
			w.cfg.Name,
			ar.Request.Namespace,
			helpers.GroupVersionResourceToString(ar.Request.Resource),
			ar.Request.Operation,
			metrics.ValidatingReviewKind)
	}
}

func (w *staticWebhook) observeAdmissionReviewDuration(ar *admissionv1beta1.AdmissionReview, start time.Time) {
	w.mRecorder.ObserveAdmissionReviewDuration(
		w.cfg.Name,
		ar.Request.Namespace,
		helpers.GroupVersionResourceToString(ar.Request.Resource),
		ar.Request.Operation,
		metrics.ValidatingReviewKind,
		start)
}
