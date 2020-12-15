package validating_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/slok/kubewebhook/pkg/model"
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
	return validating.ValidatorFunc(func(_ context.Context, _ *model.AdmissionReview, _ metav1.Object) (*validating.ValidatorResult, error) {
		return &validating.ValidatorResult{
			Valid:   valid,
			Message: message,
		}, nil
	})
}

func TestPodAdmissionReviewValidation(t *testing.T) {
	tests := map[string]struct {
		cfg         validating.WebhookConfig
		validator   validating.Validator
		review      model.AdmissionReview
		expResponse model.AdmissionResponse
		expErr      bool
	}{
		"A webhook review with error should return an error.": {
			cfg: validating.WebhookConfig{ID: "test", Obj: &corev1.Pod{}},
			validator: validating.ValidatorFunc(func(_ context.Context, _ *model.AdmissionReview, _ metav1.Object) (*validating.ValidatorResult, error) {
				return nil, fmt.Errorf("wanted error")
			}),
			review: model.AdmissionReview{ID: "test", NewObjectRaw: getPodJSON()},
			expResponse: &model.ValidatingAdmissionResponse{
				ID:      "test",
				Allowed: true,
			},
			expErr: true,
		},

		"A static webhook review of a Pod with a valid validator result should return allowed.": {
			cfg:       validating.WebhookConfig{ID: "test", Obj: &corev1.Pod{}},
			validator: getFakeValidator(true, ""),
			review:    model.AdmissionReview{ID: "test", NewObjectRaw: getPodJSON()},
			expResponse: &model.ValidatingAdmissionResponse{
				ID:      "test",
				Allowed: true,
			},
		},

		"A static webhook review of a Pod with a invalid validator result should return not allowed.": {
			cfg:       validating.WebhookConfig{ID: "test", Obj: &corev1.Pod{}},
			validator: getFakeValidator(false, "invalid test chain"),
			review:    model.AdmissionReview{ID: "test", NewObjectRaw: getPodJSON()},
			expResponse: &model.ValidatingAdmissionResponse{
				ID:      "test",
				Allowed: false,
				Message: "invalid test chain",
			},
		},

		"A static webhook review of a delete operation on a Pod should allow.": {
			cfg: validating.WebhookConfig{ID: "test", Obj: &corev1.Pod{}},
			validator: validating.ValidatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
				// Just a check to validate that is unstructured.
				pod, ok := obj.(*corev1.Pod)
				if !ok {
					return nil, fmt.Errorf("not unstructured")
				}

				// Validate.
				_, ok = pod.Labels["test1"]
				return &validating.ValidatorResult{
					Valid: ok,
				}, nil
			}),
			review: model.AdmissionReview{
				ID:           "test",
				Operation:    model.OperationDelete,
				OldObjectRaw: getPodJSON(),
			},
			expResponse: &model.ValidatingAdmissionResponse{
				ID:      "test",
				Allowed: true,
			},
		},

		"A dynamic webhook review of a Pod with a valid validator result should return allowed.": {
			cfg:       validating.WebhookConfig{ID: "test"},
			validator: getFakeValidator(true, ""),
			review:    model.AdmissionReview{ID: "test", NewObjectRaw: getPodJSON()},
			expResponse: &model.ValidatingAdmissionResponse{
				ID:      "test",
				Allowed: true,
			},
		},

		"A dynamic webhook review of a Pod with a invalid validator result should return not allowed.": {
			cfg:       validating.WebhookConfig{ID: "test"},
			validator: getFakeValidator(false, "invalid test chain"),
			review:    model.AdmissionReview{ID: "test", NewObjectRaw: getPodJSON()},
			expResponse: &model.ValidatingAdmissionResponse{
				ID:      "test",
				Allowed: false,
				Message: "invalid test chain",
			},
		},

		"A dynamic webhook review of a an unknown type should check that a label is present.": {
			cfg: validating.WebhookConfig{ID: "test"},
			validator: validating.ValidatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
				// Just a check to validate that is unstructured.
				if _, ok := obj.(runtime.Unstructured); !ok {
					return nil, fmt.Errorf("not unstructured")
				}

				// Validate.
				labels := obj.GetLabels()
				_, ok := labels["test1"]
				return &validating.ValidatorResult{
					Valid: ok,
				}, nil
			}),
			review: model.AdmissionReview{
				ID: "test",
				NewObjectRaw: []byte(`
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
			expResponse: &model.ValidatingAdmissionResponse{
				ID:      "test",
				Allowed: true,
			},
		},

		"A dynamic webhook review of a delete operation on a unknown type should check that a label is present.": {
			cfg: validating.WebhookConfig{ID: "test"},
			validator: validating.ValidatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
				// Just a check to validate that is unstructured.
				if _, ok := obj.(runtime.Unstructured); !ok {
					return nil, fmt.Errorf("not unstructured")
				}

				// Validate.
				labels := obj.GetLabels()
				_, ok := labels["test1"]
				return &validating.ValidatorResult{
					Valid: ok,
				}, nil
			}),
			review: model.AdmissionReview{
				ID:        "test",
				Operation: model.OperationDelete,
				OldObjectRaw: []byte(`
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
			expResponse: &model.ValidatingAdmissionResponse{
				ID:      "test",
				Allowed: true,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Execute.
			test.cfg.Validator = test.validator
			wh, err := validating.NewWebhook(test.cfg)
			require.NoError(err)
			gotResponse, err := wh.Review(context.TODO(), test.review)

			// Check.
			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(test.expResponse, gotResponse)
			}

		})
	}
}

func getRandomValidator() validating.Validator {
	return validating.ValidatorFunc(func(_ context.Context, _ *model.AdmissionReview, _ metav1.Object) (*validating.ValidatorResult, error) {
		valid := time.Now().Nanosecond()%2 == 0
		return &validating.ValidatorResult{Valid: valid}, nil
	})
}

func BenchmarkPodAdmissionReviewValidation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cfg := validating.WebhookConfig{
			ID:        "test",
			Obj:       &corev1.Pod{},
			Validator: getRandomValidator(),
		}
		wh, _ := validating.NewWebhook(cfg)

		ar := model.AdmissionReview{ID: "test", NewObjectRaw: getPodJSON()}
		wh.Review(context.TODO(), ar)
	}
}
