package http

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"

	"github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/webhook"
)

var (
	runtimeScheme = func() *runtime.Scheme {
		r := runtime.NewScheme()
		r.AddKnownTypes(admissionv1beta1.SchemeGroupVersion, &admissionv1beta1.AdmissionReview{})
		r.AddKnownTypes(admissionv1.SchemeGroupVersion, &admissionv1.AdmissionReview{})
		return r
	}()
	codecs       = serializer.NewCodecFactory(runtimeScheme)
	deserializer = codecs.UniversalDeserializer()
)

// MustHandlerFor it's the same as HandleFor but will panic instead of returning
// a error.
func MustHandlerFor(webhook webhook.Webhook) http.Handler {
	h, err := HandlerFor(webhook)
	if err != nil {
		panic(err)
	}
	return h
}

// HandlerFor returns a new http.Handler ready to handle admission reviews using a
// a webhook.
func HandlerFor(webhook webhook.Webhook) (http.Handler, error) {
	if webhook == nil {
		return nil, fmt.Errorf("webhook can't be nil")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get webhook body with the admission review.
		var body []byte
		if r.Body != nil {
			if data, err := ioutil.ReadAll(r.Body); err == nil {
				body = data
			}
		}
		if len(body) == 0 {
			http.Error(w, "no body found", http.StatusBadRequest)
			return
		}

		ar, err := requestBodyToModelReview(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Webhook execution logic. This is how we are dealing with the different responses:
		// |                        | HTTP Code             | status.Code | status.Status | status.Message |
		// |------------------------|-----------------------| ------------|---------------|----------------|
		// | Validating Allowed     | 200                   | -           | -             | -              |
		// | Validating not allowed | 200                   | 400         | Failure       | Custom message |
		// | Mutating mutation      | 200                   | -           | -             | -              |
		// | Mutating no mutation   | 200                   | -           | -             | -              |
		// | Err                    | 500                   | -           | Failure       | Err string     |
		admissionResp, err := webhook.Review(ctx, *ar)
		if err != nil {
			errResp, err := errorToJSON(*ar, err)
			if err != nil {
				http.Error(w, fmt.Sprintf("could not marshall status error on admission response: %v", err), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write(errResp); err != nil {
				http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
			}
			return
		}

		// Create the review response.
		resp, err := modelResponseToJSON(*ar, admissionResp)
		if err != nil {
			errResp, err := errorToJSON(*ar, err)
			if err != nil {
				http.Error(w, fmt.Sprintf("could not marshall status error on admission response: %v", err), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write(errResp); err != nil {
				http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if _, err := w.Write(resp); err != nil {
			http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
		}
	}), nil
}

func requestBodyToModelReview(body []byte) (*model.AdmissionReview, error) {
	kubeReview, _, err := deserializer.Decode(body, nil, nil)
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

func modelResponseToJSON(review model.AdmissionReview, resp model.AdmissionResponse) (data []byte, err error) {
	switch r := resp.(type) {
	case *model.ValidatingAdmissionResponse:
		return validatingModelResponseToJSON(review, r)
	case *model.MutatingAdmissionResponse:
		return mutatingModelResponseToJSON(review, r)
	default:
		return nil, fmt.Errorf("unknown webhook response type")
	}
}

func validatingModelResponseToJSON(review model.AdmissionReview, resp *model.ValidatingAdmissionResponse) (data []byte, err error) {
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
		// TODO(slok): Log warnings being used with v1beta1.
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

func mutatingModelResponseToJSON(review model.AdmissionReview, resp *model.MutatingAdmissionResponse) (data []byte, err error) {
	switch review.OriginalAdmissionReview.(type) {
	case *admissionv1beta1.AdmissionReview:
		// TODO(slok): Log warnings being used with v1beta1.
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

func errorToJSON(review model.AdmissionReview, err error) ([]byte, error) {
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
