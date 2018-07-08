package validating_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/webhook/validating"
)

func getPodJSON() []byte {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testPod",
			Namespace: "testNS",
		},
	}
	bs, _ := json.Marshal(pod)
	return bs
}

func getFakeValidator(valid bool, message string) validating.Validator {
	return validating.ValidatorFunc(func(_ context.Context, _ metav1.Object) (bool, validating.ValidatorResult, error) {
		res := validating.ValidatorResult{
			Valid:   valid,
			Message: message,
		}
		return false, res, nil
	})
}

func TestPodAdmissionReviewValidation(t *testing.T) {
	tests := []struct {
		name        string
		validator   validating.Validator
		review      *admissionv1beta1.AdmissionReview
		expResponse *admissionv1beta1.AdmissionResponse
	}{
		{
			name:      "A review of a Pod with a valid validator result should return allowed.",
			validator: getFakeValidator(true, "valid test chain"),
			review: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID: "test",
					Object: runtime.RawExtension{
						Raw: getPodJSON(),
					},
				},
			},
			expResponse: &admissionv1beta1.AdmissionResponse{
				UID:     "test",
				Allowed: true,
				Result: &metav1.Status{
					Status:  metav1.StatusSuccess,
					Message: "valid test chain",
				},
			},
		},
		{
			name:      "A review of a Pod with a invalid validator result should return not allowed.",
			validator: getFakeValidator(false, "invalid test chain"),
			review: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID: "test",
					Object: runtime.RawExtension{
						Raw: getPodJSON(),
					},
				},
			},
			expResponse: &admissionv1beta1.AdmissionResponse{
				UID:     "test",
				Allowed: false,
				Result: &metav1.Status{
					Status:  metav1.StatusSuccess,
					Message: "invalid test chain",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			wh, err := validating.NewWebhook(test.validator, &corev1.Pod{}, log.Dummy)
			require.NoError(err)
			gotResponse := wh.Review(test.review)

			assert.Equal(test.expResponse, gotResponse)
		})
	}
}

func getRandomValidator() validating.Validator {
	return validating.ValidatorFunc(func(_ context.Context, _ metav1.Object) (bool, validating.ValidatorResult, error) {
		valid := time.Now().Nanosecond()%2 == 0
		return false, validating.ValidatorResult{Valid: valid}, nil
	})
}

func BenchmarkPodAdmissionReviewValidation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ar := &admissionv1beta1.AdmissionReview{
			Request: &admissionv1beta1.AdmissionRequest{
				UID: "test",
				Object: runtime.RawExtension{
					Raw: getPodJSON(),
				},
			},
		}
		wh, _ := validating.NewWebhook(getRandomValidator(), &corev1.Pod{}, log.Dummy)
		wh.Review(ar)
	}
}
