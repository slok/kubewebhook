package mutating

import (
	"github.com/slok/kubewebhook/v2/pkg/log"
	"github.com/slok/kubewebhook/v2/pkg/webhook"
	"github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
)

// NewPodWebhook returns a new pod mutating webhook.
func NewPodWebhook(labels map[string]string, logger log.Logger) (webhook.Webhook, error) {
	// Create mutators.
	mutators := []mutating.Mutator{
		&podLabelMutator{labels: labels, logger: logger},
		&lantencyMutator{maxLatencyMS: 20},
		&lantencyMutator{maxLatencyMS: 120},
		&lantencyMutator{maxLatencyMS: 300},
		&lantencyMutator{maxLatencyMS: 10},
		&lantencyMutator{maxLatencyMS: 175},
		&lantencyMutator{maxLatencyMS: 80},
		&lantencyMutator{maxLatencyMS: 10},
	}

	return mutating.NewWebhook(mutating.WebhookConfig{
		ID:      "multiwebhook-podMutator",
		Obj:     &corev1.Pod{},
		Mutator: mutating.NewChain(logger, mutators...),
		Logger:  logger,
	})
}
