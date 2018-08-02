package mutating

import (
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
)

// NewPodWebhook returns a new pod mutating webhook.
func NewPodWebhook(labels map[string]string, ot opentracing.Tracer, mrec metrics.Recorder, logger log.Logger) (webhook.Webhook, error) {
	// Create mutators.
	mutators := []mutating.Mutator{
		mutating.TraceMutator(ot, "podLabelMutator", &podLabelMutator{labels: labels, logger: logger}),
		mutating.TraceMutator(ot, "latencyMutator20ms", &lantencyMutator{maxLatencyMS: 20}),
		mutating.TraceMutator(ot, "latencyMutator120ms", &lantencyMutator{maxLatencyMS: 120}),
		mutating.TraceMutator(ot, "latencyMutator300ms", &lantencyMutator{maxLatencyMS: 300}),
		mutating.TraceMutator(ot, "latencyMutator10ms", &lantencyMutator{maxLatencyMS: 10}),
		mutating.TraceMutator(ot, "latencyMutator175ms", &lantencyMutator{maxLatencyMS: 175}),
		mutating.TraceMutator(ot, "latencyMutator80ms", &lantencyMutator{maxLatencyMS: 80}),
		mutating.TraceMutator(ot, "latencyMutator10ms", &lantencyMutator{maxLatencyMS: 10}),
	}

	mc := mutating.NewChain(logger, mutators...)
	cfg := mutating.WebhookConfig{
		Name: "multiwebhook-podMutator",
		Obj:  &corev1.Pod{},
	}

	return mutating.NewWebhook(cfg, mc, ot, mrec, logger)
}
