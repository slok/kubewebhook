package mutating

import (
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhwebhook "github.com/slok/kubewebhook/v2/pkg/webhook"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
)

// NewPodWebhook returns a new pod mutating webhook.
func NewPodWebhook(labels map[string]string, logger kwhlog.Logger) (kwhwebhook.Webhook, error) {
	// Create mutators.
	mutators := []kwhmutating.Mutator{
		&podLabelMutator{labels: labels, logger: logger},
		&lantencyMutator{maxLatencyMS: 20},
		&lantencyMutator{maxLatencyMS: 120},
		&lantencyMutator{maxLatencyMS: 300},
		&lantencyMutator{maxLatencyMS: 10},
		&lantencyMutator{maxLatencyMS: 175},
		&lantencyMutator{maxLatencyMS: 80},
		&lantencyMutator{maxLatencyMS: 10},
	}

	return kwhmutating.NewWebhook(kwhmutating.WebhookConfig{
		ID:      "multiwebhook-podMutator",
		Obj:     &corev1.Pod{},
		Mutator: kwhmutating.NewChain(logger, mutators...),
		Logger:  logger,
	})
}
