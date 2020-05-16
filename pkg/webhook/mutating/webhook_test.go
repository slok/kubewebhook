package mutating_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
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
			Annotations: map[string]string{
				"key1": "val1",
				"key2": "val2",
				"key3": "val3",
				"key4": "val4",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container1",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("10m"),
							corev1.ResourceMemory: resource.MustParse("10Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("100Mi"),
						},
					},
				},
				{
					Name: "container2",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("30m"),
							corev1.ResourceMemory: resource.MustParse("30Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("70m"),
							corev1.ResourceMemory: resource.MustParse("70Mi"),
						},
					},
				},
			},
		},
	}
	bs, _ := json.Marshal(pod)
	return bs
}

func getPodNSMutator(ns string) mutating.Mutator {
	return mutating.MutatorFunc(func(_ context.Context, obj metav1.Object) (bool, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return true, fmt.Errorf("not a pod")
		}

		pod.Namespace = ns

		return false, nil
	})
}

func getPodAnnotationsReplacerMutator(annotations map[string]string) mutating.Mutator {
	return mutating.MutatorFunc(func(_ context.Context, obj metav1.Object) (bool, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return true, fmt.Errorf("not a pod")
		}

		pod.Annotations = annotations

		return false, nil
	})
}

func getPodResourceLimitDeletorMutator() mutating.Mutator {
	return mutating.MutatorFunc(func(_ context.Context, obj metav1.Object) (bool, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return true, fmt.Errorf("not a pod")
		}

		for idx := range pod.Spec.Containers {
			c := pod.Spec.Containers[idx]
			c.Resources.Limits = nil
			pod.Spec.Containers[idx] = c
		}

		return false, nil
	})
}

func TestPodAdmissionReviewMutation(t *testing.T) {
	tests := map[string]struct {
		cfg      mutating.WebhookConfig
		mutator  mutating.Mutator
		review   *admissionv1beta1.AdmissionReview
		expPatch []string
	}{
		"A static webhook review of a Pod with an ns mutator should mutate the ns.": {
			cfg:     mutating.WebhookConfig{Name: "test", Obj: &corev1.Pod{}},
			mutator: getPodNSMutator("myChangedNS"),
			review: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID: "test",
					Object: runtime.RawExtension{
						Raw: getPodJSON(),
					},
				},
			},
			expPatch: []string{
				`{"op":"replace","path":"/metadata/namespace","value":"myChangedNS"}`,
			},
		},

		"A static webhook review of a Pod with an annotations mutator should mutate the annotations.": {
			cfg: mutating.WebhookConfig{Name: "test", Obj: &corev1.Pod{}},
			mutator: getPodAnnotationsReplacerMutator(map[string]string{
				"key1": "val1_mutated",
				"key2": "val2",
				"key4": "val4",
				"key5": "val5",
			}),
			review: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID: "test",
					Object: runtime.RawExtension{
						Raw: getPodJSON(),
					},
				},
			},
			expPatch: []string{
				`{"op":"replace","path":"/metadata/annotations/key1","value":"val1_mutated"}`,
				`{"op":"add","path":"/metadata/annotations/key5","value":"val5"}`,
				`{"op":"remove","path":"/metadata/annotations/key3"}`,
			},
		},

		"A static webhook review of a Pod with an limit deletion mutator should delete the limi resources from a pod.": {
			cfg:     mutating.WebhookConfig{Name: "test", Obj: &corev1.Pod{}},
			mutator: getPodResourceLimitDeletorMutator(),
			review: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID: "test",
					Object: runtime.RawExtension{
						Raw: getPodJSON(),
					},
				},
			},
			expPatch: []string{
				`{"op":"remove","path":"/spec/containers/0/resources/limits"}`,
				`{"op":"remove","path":"/spec/containers/1/resources/limits"}`,
			},
		},

		"A static webhook review of delete operation in a Pod should mutate the pod correctly.": {
			cfg:     mutating.WebhookConfig{Name: "test", Obj: &corev1.Pod{}},
			mutator: getPodResourceLimitDeletorMutator(),
			review: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Operation: admissionv1beta1.Delete,
					UID:       "test",
					OldObject: runtime.RawExtension{
						Raw: getPodJSON(),
					},
				},
			},
			expPatch: []string{
				`{"op":"remove","path":"/spec/containers/0/resources/limits"}`,
				`{"op":"remove","path":"/spec/containers/1/resources/limits"}`,
			},
		},

		"A dynamic webhook review of a Pod with an ns mutator should mutate the ns.": {
			cfg:     mutating.WebhookConfig{Name: "test"},
			mutator: getPodNSMutator("myChangedNS"),
			review: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID: "test",
					Object: runtime.RawExtension{
						Raw: getPodJSON(),
					},
				},
			},
			expPatch: []string{
				`{"op":"replace","path":"/metadata/namespace","value":"myChangedNS"}`,
			},
		},

		"A dynamic webhook review of a Pod with an annotations mutator should mutate the annotations.": {
			cfg: mutating.WebhookConfig{Name: "test"},
			mutator: getPodAnnotationsReplacerMutator(map[string]string{
				"key1": "val1_mutated",
				"key2": "val2",
				"key4": "val4",
				"key5": "val5",
			}),
			review: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID: "test",
					Object: runtime.RawExtension{
						Raw: getPodJSON(),
					},
				},
			},
			expPatch: []string{
				`{"op":"replace","path":"/metadata/annotations/key1","value":"val1_mutated"}`,
				`{"op":"add","path":"/metadata/annotations/key5","value":"val5"}`,
				`{"op":"remove","path":"/metadata/annotations/key3"}`,
			},
		},

		"A dynamic webhook review of a Pod with an limit deletion mutator should delete the limi resources from a pod.": {
			cfg:     mutating.WebhookConfig{Name: "test"},
			mutator: getPodResourceLimitDeletorMutator(),
			review: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID: "test",
					Object: runtime.RawExtension{
						Raw: getPodJSON(),
					},
				},
			},
			expPatch: []string{
				`{"op":"remove","path":"/spec/containers/0/resources/limits"}`,
				`{"op":"remove","path":"/spec/containers/1/resources/limits"}`,
			},
		},

		"A dynamic webhook review of a an unknown type should be able to mutate with the common object attributes (check unstructured object mutation).": {
			cfg: mutating.WebhookConfig{Name: "test"},
			mutator: mutating.MutatorFunc(func(_ context.Context, obj metav1.Object) (bool, error) {
				// Just a check to validate that is unstructured.
				if _, ok := obj.(runtime.Unstructured); !ok {
					return true, fmt.Errorf("not unstructured")
				}

				// Mutate.
				labels := obj.GetLabels()
				if labels == nil {
					labels = map[string]string{}
				}
				labels["test1"] = "mutated-value1"
				labels["test2"] = "mutated-value2"
				obj.SetLabels(labels)

				return false, nil
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
								},
								"annotations":{
									"key1":"val1",
									"key2":"val2"
								}
							},
							"spec": {
								"n": 42 
							}
						}`),
					},
				},
			},
			expPatch: []string{
				`{"op":"replace","path":"/metadata/labels/test1","value":"mutated-value1"}`,
				`{"op":"add","path":"/metadata/labels/test2","value":"mutated-value2"}`,
			},
		},

		"A dynamic webhook delete operation review of an unknown type should be able to mutate with the common object attributes (check unstructured object mutation).": {
			cfg: mutating.WebhookConfig{Name: "test"},
			mutator: mutating.MutatorFunc(func(_ context.Context, obj metav1.Object) (bool, error) {
				// Just a check to validate that is unstructured.
				if _, ok := obj.(runtime.Unstructured); !ok {
					return true, fmt.Errorf("not unstructured")
				}

				// Mutate.
				labels := obj.GetLabels()
				if labels == nil {
					labels = map[string]string{}
				}
				labels["test1"] = "mutated-value1"
				labels["test2"] = "mutated-value2"
				obj.SetLabels(labels)

				return false, nil
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
								},
								"annotations":{
									"key1":"val1",
									"key2":"val2"
								}
							},
							"spec": {
								"n": 42 
							}
						}`),
					},
				},
			},
			expPatch: []string{
				`{"op":"replace","path":"/metadata/labels/test1","value":"mutated-value1"}`,
				`{"op":"add","path":"/metadata/labels/test2","value":"mutated-value2"}`,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			wh, err := mutating.NewWebhook(test.cfg, test.mutator, nil, nil, log.Dummy)
			assert.NoError(err)

			gotResponse := wh.Review(context.TODO(), test.review)

			// Check uid, allowed and patch
			assert.True(gotResponse.Allowed)
			assert.Equal(test.review.Request.UID, gotResponse.UID)
			gotPatch := string(gotResponse.Patch)
			for _, expPatchOp := range test.expPatch {
				assert.Contains(gotPatch, expPatchOp)
			}
		})
	}
}

func BenchmarkPodAdmissionReviewMutation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mutator := getPodNSMutator("myChangedNS")
		ar := &admissionv1beta1.AdmissionReview{
			Request: &admissionv1beta1.AdmissionRequest{
				UID: "test",
				Object: runtime.RawExtension{
					Raw: getPodJSON(),
				},
			},
		}

		cfg := mutating.WebhookConfig{
			Name: "test",
			Obj:  &corev1.Pod{},
		}
		wh, err := mutating.NewWebhook(cfg, mutator, nil, nil, log.Dummy)
		assert.NoError(b, err)
		wh.Review(context.TODO(), ar)
	}
}
