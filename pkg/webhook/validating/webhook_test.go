package validating_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	"github.com/slok/kubewebhook/pkg/webhook/validating"
)

func getPodJSON() []byte {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testPod",
			Namespace: "testNS",
			Labels: map[string]string{
				"test1": "value1",
			},
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
	tests := map[string]struct {
		cfg         validating.WebhookConfig
		validator   validating.Validator
		review      *admissionv1beta1.AdmissionReview
		expResponse *admissionv1beta1.AdmissionResponse
	}{
		"A static webhook review of a Pod with a valid validator result should return allowed.": {
			cfg:       validating.WebhookConfig{Name: "test", Obj: &corev1.Pod{}},
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

		"A static webhook review of a Pod with a invalid validator result should return not allowed.": {
			cfg:       validating.WebhookConfig{Name: "test", Obj: &corev1.Pod{}},
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
					Message: "invalid test chain",
				},
			},
		},

		"A static webhook review of a delete operation on a Pod should allow.": {
			cfg: validating.WebhookConfig{Name: "test", Obj: &corev1.Pod{}},
			validator: validating.ValidatorFunc(func(_ context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
				// Just a check to validate that is unstructured.
				pod, ok := obj.(*corev1.Pod)
				if !ok {
					return true, validating.ValidatorResult{}, fmt.Errorf("not unstructured")
				}

				// Validate.
				_, ok = pod.Labels["test1"]
				return false, validating.ValidatorResult{
					Valid:   ok,
					Message: "label present",
				}, nil
			}),
			review: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Operation: admissionv1beta1.Delete,
					UID:       "test",
					OldObject: runtime.RawExtension{
						Raw: getPodJSON(),
					},
				},
			},
			expResponse: &admissionv1beta1.AdmissionResponse{
				UID:     "test",
				Allowed: true,
				Result: &metav1.Status{
					Status:  metav1.StatusSuccess,
					Message: "label present",
				},
			},
		},

		"A dynamic webhook review of a Pod with a valid validator result should return allowed.": {
			cfg:       validating.WebhookConfig{Name: "test"},
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

		"A dynamic webhook review of a Pod with a invalid validator result should return not allowed.": {
			cfg:       validating.WebhookConfig{Name: "test"},
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
					Message: "invalid test chain",
				},
			},
		},

		"A dynamic webhook review of a an unknown type should check that a label is present.": {
			cfg: validating.WebhookConfig{Name: "test"},
			validator: validating.ValidatorFunc(func(_ context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
				// Just a check to validate that is unstructured.
				if _, ok := obj.(runtime.Unstructured); !ok {
					return true, validating.ValidatorResult{}, fmt.Errorf("not unstructured")
				}

				// Validate.
				labels := obj.GetLabels()
				_, ok := labels["test1"]
				return false, validating.ValidatorResult{
					Valid:   ok,
					Message: "label present",
				}, nil
			}),
			review: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID: "test",
					Object: runtime.RawExtension{
						Raw: []byte(`
						{
							"kind": "whatever",
							"apiVersion": "v42",
							"metadata": {
								"name":"something",
								"namespace":"someplace",
								"labels": {
									"test1": "value1"
								}
							}
						}`),
					},
				},
			},
			expResponse: &admissionv1beta1.AdmissionResponse{
				UID:     "test",
				Allowed: true,
				Result: &metav1.Status{
					Status:  metav1.StatusSuccess,
					Message: "label present",
				},
			},
		},

		"A dynamic webhook review of a delete operation on a unknown type should check that a label is present.": {
			cfg: validating.WebhookConfig{Name: "test"},
			validator: validating.ValidatorFunc(func(_ context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
				// Just a check to validate that is unstructured.
				if _, ok := obj.(runtime.Unstructured); !ok {
					return true, validating.ValidatorResult{}, fmt.Errorf("not unstructured")
				}

				// Validate.
				labels := obj.GetLabels()
				_, ok := labels["test1"]
				return false, validating.ValidatorResult{
					Valid:   ok,
					Message: "label present",
				}, nil
			}),
			review: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID:       "test",
					Operation: admissionv1beta1.Delete,
					OldObject: runtime.RawExtension{
						Raw: []byte(`
						{
							"kind": "whatever",
							"apiVersion": "v42",
							"metadata": {
								"name":"something",
								"namespace":"someplace",
								"labels": {
									"test1": "value1"
								}
							}
						}`),
					},
				},
			},
			expResponse: &admissionv1beta1.AdmissionResponse{
				UID:     "test",
				Allowed: true,
				Result: &metav1.Status{
					Status:  metav1.StatusSuccess,
					Message: "label present",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			wh, err := validating.NewWebhook(test.cfg, test.validator, nil, nil, log.Dummy)
			require.NoError(err)
			gotResponse := wh.Review(context.TODO(), test.review)

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

		cfg := validating.WebhookConfig{
			Name: "test",
			Obj:  &corev1.Pod{},
		}

		wh, _ := validating.NewWebhook(cfg, getRandomValidator(), nil, metrics.Dummy, log.Dummy)
		wh.Review(context.TODO(), ar)
	}
}
