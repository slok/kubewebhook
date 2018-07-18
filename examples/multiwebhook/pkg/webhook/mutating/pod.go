package mutating

import (
	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
)

// NewPodWebhook returns a new pod mutating webhook.
func NewPodWebhook(labels map[string]string, mrec metrics.Recorder, logger log.Logger) (webhook.Webhook, error) {

	// Create mutators.
	mut := &podLabelMutator{
		labels: labels,
		logger: logger,
	}

	cfg := mutating.WebhookConfig{
		Name: "multiwebhook-podMutator",
		Obj:  &corev1.Pod{},
	}

	return mutating.NewWebhook(cfg, mut, mrec, logger)
}
