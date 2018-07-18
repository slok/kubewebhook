package mutating

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/pkg/log"
)

// podLabelMutator will add labels to a pod. Satisfies mutating.Mutator interface.
type podLabelMutator struct {
	labels map[string]string
	logger log.Logger
}

func (p *podLabelMutator) Mutate(_ context.Context, obj metav1.Object) (bool, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		// If not a pod just continue the mutation chain(if there is one) and don't do nothing.
		return false, nil
	}

	// Mutate our object with the required annotations.
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	for k, v := range p.labels {
		pod.Labels[k] = v
	}

	return false, nil
}
