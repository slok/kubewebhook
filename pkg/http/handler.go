package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"

	"github.com/slok/kubewebhook/v2/pkg/log"
	"github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/tracing"
	"github.com/slok/kubewebhook/v2/pkg/webhook"
)

var (
	admissionReviewDeserializer = func() runtime.Decoder {
		r := runtime.NewScheme()
		r.AddKnownTypes(admissionv1beta1.SchemeGroupVersion, &admissionv1beta1.AdmissionReview{})
		r.AddKnownTypes(admissionv1.SchemeGroupVersion, &admissionv1.AdmissionReview{})

		codecs := serializer.NewCodecFactory(r)

		return codecs.UniversalDeserializer()
	}()
)

// MustHandlerFor it's the same as HandleFor but will panic instead of returning
// a error.
func MustHandlerFor(config HandlerConfig) http.Handler {
	h, err := HandlerFor(config)
	if err != nil {
		panic(err)
	}
	return h
}

// HandlerConfig is the configuration for the webhook handlers.
type HandlerConfig struct {
	Webhook webhook.Webhook
	Logger  log.Logger
	Tracer  tracing.Tracer
}

func (c *HandlerConfig) defaults() error {
	if c.Webhook == nil {
		return fmt.Errorf("webhook can't be nil")
	}

	if c.Logger == nil {
		c.Logger = log.Noop
	}
	c.Logger = c.Logger.WithValues(log.Kv{"svc": "http.Handler"})

	if c.Tracer == nil {
		c.Tracer = tracing.Noop
	}
	c.Tracer = c.Tracer.WithValues(map[string]interface{}{"svc": "http.Handler"})

	return nil
}

// HandlerFor returns a new http.Handler ready to handle admission reviews using a
// a webhook.
func HandlerFor(config HandlerConfig) (http.Handler, error) {
	err := config.defaults()
	if err != nil {
		return nil, fmt.Errorf("handler invalid configuration: %w", err)
	}

	h := config.Tracer.TraceHTTPHandler("webhookHTTPHandler", handler{
		webhook: config.Webhook,
		logger:  config.Logger,
		tracer:  config.Tracer,
	})

	return h, nil
}

type handler struct {
	webhook webhook.Webhook
	logger  log.Logger
	tracer  tracing.Tracer
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	t0 := time.Now()

	// Get webhook body with the admission review.
	var body []byte
	if r.Body != nil {
		if data, err := configReader(r); err == nil {
			body = data
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			h.logger.Errorf(err.Error())
			return
		}
	}
	if len(body) == 0 {
		http.Error(w, "no body found", http.StatusBadRequest)
		h.logger.Errorf("no body found")
		return
	}

	ar, err := h.requestBodyToModelReview(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Errorf("could not parse body to model review: %s", err)
		return
	}

	// Setup log data on context.
	ctx = h.logger.SetValuesOnCtx(ctx, log.Kv{
		"webhook-id":   h.webhook.ID(),
		"webhook-kind": h.webhook.Kind(),
		"request-id":   ar.ID,
		"op":           ar.Operation,
		"wh-version":   ar.Version,
		"dry-run":      ar.DryRun,
		"kind":         strings.Trim(strings.Join([]string{ar.RequestGVK.Group, ar.RequestGVK.Version, ar.RequestGVK.Kind}, "/"), " /"),
		"ns":           ar.Namespace,
		"name":         ar.Name,
		"path":         r.URL.Path,
		"trace-id":     h.tracer.TraceID(ctx),
	})
	logger := h.logger.WithCtxValues(ctx)

	// Webhook execution logic. This is how we are dealing with the different responses:
	// |                        | HTTP Code             | status.Code | status.Status | status.Message |
	// |------------------------|-----------------------| ------------|---------------|----------------|
	// | Validating Allowed     | 200                   | -           | -             | -              |
	// | Validating not allowed | 200                   | 400         | Failure       | Custom message |
	// | Mutating mutation      | 200                   | -           | -             | -              |
	// | Mutating no mutation   | 200                   | -           | -             | -              |
	// | Err                    | 500                   | -           | Failure       | Err string     |
	admissionResp, err := h.webhook.Review(ctx, *ar)
	if err != nil {
		logger.Errorf("Admission review error: %s", err)

		errResp, err := h.errorToJSON(*ar, err)
		if err != nil {
			msg := fmt.Sprintf("could not marshall status error on admission response: %v", err)
			http.Error(w, msg, http.StatusInternalServerError)
			logger.Errorf(msg)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write(errResp); err != nil {
			msg := fmt.Sprintf("could not write response: %v", err)
			http.Error(w, msg, http.StatusInternalServerError)
			logger.Errorf(msg)
			return
		}

		return
	}

	// Create the review response.
	resp, err := h.modelResponseToJSON(ctx, *ar, admissionResp)
	if err != nil {
		logger.Errorf("Could not map model response to JSON: %s", err)

		errResp, err := h.errorToJSON(*ar, err)
		if err != nil {
			msg := fmt.Sprintf("could not marshall status error on admission response: %v", err)
			http.Error(w, msg, http.StatusInternalServerError)
			logger.Errorf(msg)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write(errResp); err != nil {
			msg := fmt.Sprintf("could not write response: %v", err)
			http.Error(w, msg, http.StatusInternalServerError)
			logger.Errorf(msg)
			return
		}

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write(resp); err != nil {
		msg := fmt.Sprintf("could not write response: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		logger.Errorf(msg)
		return
	}

	logger.WithValues(log.Kv{
		"duration": time.Since(t0),
	}).Infof("Admission review request handled")
}
func (h handler) requestBodyToModelReview(body []byte) (*model.AdmissionReview, error) {
	kubeReview, _, err := admissionReviewDeserializer.Decode(body, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("could not decode the admission review from the request: %w", err)
	}

	switch ar := kubeReview.(type) {
	case *admissionv1beta1.AdmissionReview:
		res := model.NewAdmissionReviewV1Beta1(ar)
		return &res, nil
	case *admissionv1.AdmissionReview:
		res := model.NewAdmissionReviewV1(ar)
		return &res, nil
	}

	return nil, fmt.Errorf("invalid admission review type")
}

func (h handler) modelResponseToJSON(ctx context.Context, review model.AdmissionReview, resp model.AdmissionResponse) (data []byte, err error) {
	switch r := resp.(type) {
	case *model.ValidatingAdmissionResponse:
		return h.validatingModelResponseToJSON(ctx, review, r)
	case *model.MutatingAdmissionResponse:
		return h.mutatingModelResponseToJSON(ctx, review, r)
	default:
		return nil, fmt.Errorf("unknown webhook response type")
	}
}

func (h handler) validatingModelResponseToJSON(ctx context.Context, review model.AdmissionReview, resp *model.ValidatingAdmissionResponse) (data []byte, err error) {
	// Set the satus code and result based on the validation result.
	var resultStatus *metav1.Status
	if !resp.Allowed {
		resultStatus = &metav1.Status{
			Message: resp.Message,
			Status:  metav1.StatusFailure,
			Code:    http.StatusBadRequest,
		}
	}

	switch review.OriginalAdmissionReview.(type) {
	case *admissionv1beta1.AdmissionReview:
		if len(resp.Warnings) > 0 {
			h.logger.WithCtxValues(ctx).Warningf("warnings used in a 'v1beta1' webhook")
		}

		data, err := json.Marshal(admissionv1beta1.AdmissionReview{
			TypeMeta: v1beta1AdmissionReviewTypeMeta,
			Response: &admissionv1beta1.AdmissionResponse{
				UID:     types.UID(review.ID),
				Allowed: resp.Allowed,
				Result:  resultStatus,
			},
		})
		return data, err

	case *admissionv1.AdmissionReview:
		data, err := json.Marshal(admissionv1.AdmissionReview{
			TypeMeta: v1AdmissionReviewTypeMeta,
			Response: &admissionv1.AdmissionResponse{
				UID:      types.UID(review.ID),
				Warnings: resp.Warnings,
				Allowed:  resp.Allowed,
				Result:   resultStatus,
			},
		})
		return data, err
	}

	return nil, fmt.Errorf("invalid admission response type")
}

func (h handler) mutatingModelResponseToJSON(ctx context.Context, review model.AdmissionReview, resp *model.MutatingAdmissionResponse) (data []byte, err error) {
	switch review.OriginalAdmissionReview.(type) {
	case *admissionv1beta1.AdmissionReview:
		if len(resp.Warnings) > 0 {
			h.logger.WithCtxValues(ctx).Warningf("warnings used in a 'v1beta1' webhook")
		}

		data, err := json.Marshal(admissionv1beta1.AdmissionReview{
			TypeMeta: v1beta1AdmissionReviewTypeMeta,
			Response: &admissionv1beta1.AdmissionResponse{
				UID:       types.UID(review.ID),
				PatchType: v1beta1JSONPatchType,
				Patch:     resp.JSONPatchPatch,
				Allowed:   true,
			},
		})
		return data, err

	case *admissionv1.AdmissionReview:
		data, err := json.Marshal(admissionv1.AdmissionReview{
			TypeMeta: v1AdmissionReviewTypeMeta,
			Response: &admissionv1.AdmissionResponse{
				UID:       types.UID(review.ID),
				PatchType: v1JSONPatchType,
				Patch:     resp.JSONPatchPatch,
				Allowed:   true,
				Warnings:  resp.Warnings,
			},
		})

		return data, err
	}

	return nil, fmt.Errorf("invalid admission response type")
}

func (h handler) errorToJSON(review model.AdmissionReview, err error) ([]byte, error) {
	switch review.OriginalAdmissionReview.(type) {
	case *admissionv1beta1.AdmissionReview:
		r := &admissionv1beta1.AdmissionResponse{
			UID: types.UID(review.ID),
			Result: &metav1.Status{
				Message: err.Error(),
				Status:  metav1.StatusFailure,
			},
		}

		return json.Marshal(admissionv1beta1.AdmissionReview{
			TypeMeta: v1beta1AdmissionReviewTypeMeta,
			Response: r,
		})
	case *admissionv1.AdmissionReview:
		r := &admissionv1.AdmissionResponse{
			UID: types.UID(review.ID),
			Result: &metav1.Status{
				Message: err.Error(),
				Status:  metav1.StatusFailure,
			},
		}

		return json.Marshal(admissionv1.AdmissionReview{
			TypeMeta: v1AdmissionReviewTypeMeta,
			Response: r,
		})
	}

	return nil, fmt.Errorf("invalid admission response type")
}

// MaxRequestBodyBytes represents the max size of Kubernetes objects we read. Kubernetes allows a 2x
// buffer on the max etcd size
// (https://github.com/kubernetes/kubernetes/blob/0afa569499d480df4977568454a50790891860f5/staging/src/k8s.io/apiserver/pkg/server/config.go#L362).
// We allow an additional 2x buffer, as it is still fairly cheap (6mb)
// Taken from https://github.com/istio/istio/commit/6ca5055a4db6695ef5504eabdfde3799f2ea91fd
const MaxRequestBodyBytes = int64(6 * 1024 * 1024)

// configReader is reads an HTTP request, imposing size restrictions aligned with Kubernetes limits.
func configReader(req *http.Request) ([]byte, error) {
	defer req.Body.Close()
	lr := &io.LimitedReader{
		R: req.Body,
		N: MaxRequestBodyBytes + 1,
	}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if lr.N <= 0 {
		return nil, errors.NewRequestEntityTooLargeError(fmt.Sprintf("limit is %d", MaxRequestBodyBytes))
	}
	return data, nil
}

var (
	v1beta1JSONPatchType = func() *admissionv1beta1.PatchType {
		pt := admissionv1beta1.PatchTypeJSONPatch
		return &pt
	}()
	v1JSONPatchType = func() *admissionv1.PatchType {
		pt := admissionv1.PatchTypeJSONPatch
		return &pt
	}()

	v1beta1AdmissionReviewTypeMeta = metav1.TypeMeta{
		Kind:       "AdmissionReview",
		APIVersion: "admission.k8s.io/v1beta1",
	}

	v1AdmissionReviewTypeMeta = metav1.TypeMeta{
		Kind:       "AdmissionReview",
		APIVersion: "admission.k8s.io/v1",
	}
)
