package mutating_test

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/v2/pkg/log"
	"github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
)

// PodAnnotateMutatingWebhook shows how you would create a pod mutating webhook that adds
// annotations to every pod received.
func ExampleMutator_podAnnotateMutatingWebhook() {
	// Annotations to add.
	annotations := map[string]string{
		"mutated":   "true",
		"example":   "ExamplePodAnnotateMutatingWebhook",
		"framework": "kubewebhook",
	}
	// Create our mutator that will add annotations to every pod.
	pam := mutating.MutatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*mutating.MutatorResult, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return &mutating.MutatorResult{}, nil
		}

		// Mutate our object with the required annotations.
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		for k, v := range annotations {
			pod.Annotations[k] = v
		}

		return &mutating.MutatorResult{MutatedObject: pod}, nil
	})

	// Create webhook.
	_, _ = mutating.NewWebhook(mutating.WebhookConfig{
		ID:      "podAnnotateMutatingWebhook",
		Obj:     &corev1.Pod{},
		Mutator: pam,
	})
}

// chainMutatingWebhook shows how you would create a mutator chain.
func ExampleMutator_chainMutatingWebhook() {
	fakeMut := mutating.MutatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*mutating.MutatorResult, error) {
		return &mutating.MutatorResult{}, nil
	})

	fakeMut2 := mutating.MutatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*mutating.MutatorResult, error) {
		return &mutating.MutatorResult{}, nil
	})

	fakeMut3 := mutating.MutatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*mutating.MutatorResult, error) {
		return &mutating.MutatorResult{}, nil
	})

	// Create webhook using a mutator chain.
	_, _ = mutating.NewWebhook(mutating.WebhookConfig{
		ID:      "podWebhook",
		Obj:     &corev1.Pod{},
		Mutator: mutating.NewChain(log.Dummy, fakeMut, fakeMut2, fakeMut3),
	})
}
