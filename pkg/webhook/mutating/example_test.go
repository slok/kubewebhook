package mutating_test

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
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
	pam := mutating.MutatorFunc(func(_ context.Context, obj metav1.Object) (bool, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return false, nil
		}

		// Mutate our object with the required annotations.
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		for k, v := range annotations {
			pod.Annotations[k] = v
		}

		return false, nil
	})

	// Create webhook (usage of webhook not in this example).
	mutating.NewStaticWebhook(pam, &corev1.Pod{}, log.Dummy)
}
