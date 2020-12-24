package mutating

import (
	"context"
	"math/rand"
	"time"

	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// podLabelMutator will add labels to a pod. Satisfies mutatingMutator interface.
type podLabelMutator struct {
	labels map[string]string
	logger kwhlog.Logger
}

func (m *podLabelMutator) Mutate(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		// If not a pod just continue the mutation chain(if there is one) and don't do nothing.
		return &kwhmutating.MutatorResult{}, nil
	}

	// Mutate our object with the required annotations.
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	for k, v := range m.labels {
		pod.Labels[k] = v
	}

	return &kwhmutating.MutatorResult{MutatedObject: obj}, nil
}

type lantencyMutator struct {
	maxLatencyMS int
}

func (m *lantencyMutator) Mutate(_ context.Context, _ *kwhmodel.AdmissionReview, _ metav1.Object) (*kwhmutating.MutatorResult, error) {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	ms := time.Duration(rand.Intn(m.maxLatencyMS)) * time.Millisecond
	time.Sleep(ms)
	return &kwhmutating.MutatorResult{}, nil
}
