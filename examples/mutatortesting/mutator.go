package mutatortesting

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/pkg/webhook/mutating"
)

// PodLabeler is a mutator that will set labels on the received pods.
type PodLabeler struct {
	labels map[string]string
}

// NewPodLabeler returns a new PodLabeler initialized.
func NewPodLabeler(labels map[string]string) mutating.Mutator {
	if labels == nil {
		labels = make(map[string]string)
	}
	return &PodLabeler{
		labels: labels,
	}
}

// Mutate will set the required labels on the pods. Satisfies mutating.Mutator interface.
func (p *PodLabeler) Mutate(ctx context.Context, obj metav1.Object) (bool, error) {
	pod := obj.(*corev1.Pod)

	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	for k, v := range p.labels {
		pod.Labels[k] = v
	}
	return false, nil
}
