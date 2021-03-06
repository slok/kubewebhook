package mutatortesting

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
)

// PodLabeler is a mutator that will set labels on the received pods.
type PodLabeler struct {
	labels map[string]string
}

// NewPodLabeler returns a new PodLabeler initialized.
func NewPodLabeler(labels map[string]string) kwhmutating.Mutator {
	if labels == nil {
		labels = make(map[string]string)
	}
	return &PodLabeler{
		labels: labels,
	}
}

// Mutate will set the required labels on the pods. Satisfies mutating.Mutator interface.
func (p *PodLabeler) Mutate(ctx context.Context, ar *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
	pod := obj.(*corev1.Pod)

	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	for k, v := range p.labels {
		pod.Labels[k] = v
	}
	return &kwhmutating.MutatorResult{
		MutatedObject: pod,
	}, nil
}
