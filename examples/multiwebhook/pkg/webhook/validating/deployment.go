package validating

import (
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/validating"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
)

// NewDeploymentWebhook returns a new deployment validationg webhook.
func NewDeploymentWebhook(minReplicas, maxReplicas int, ot opentracing.Tracer, mrec metrics.Recorder, logger log.Logger) (webhook.Webhook, error) {

	// Create validators.
	repVal := &deploymentReplicasValidator{
		maxReplicas: maxReplicas,
		minReplicas: minReplicas,
		logger:      logger,
	}

	vals := []validating.Validator{
		validating.TraceValidator(ot, "latencyValidator20ms", &lantencyValidator{maxLatencyMS: 20}),
		validating.TraceValidator(ot, "latencyValidator120ms", &lantencyValidator{maxLatencyMS: 120}),
		validating.TraceValidator(ot, "latencyValidator300ms", &lantencyValidator{maxLatencyMS: 300}),
		validating.TraceValidator(ot, "latencyValidator10ms", &lantencyValidator{maxLatencyMS: 10}),
		validating.TraceValidator(ot, "latencyValidator175ms", &lantencyValidator{maxLatencyMS: 175}),
		validating.TraceValidator(ot, "latencyValidator80ms", &lantencyValidator{maxLatencyMS: 80}),
		validating.TraceValidator(ot, "latencyValidator10ms", &lantencyValidator{maxLatencyMS: 10}),
		validating.TraceValidator(ot, "deploymentReplicasValidator", repVal),
	}
	valChain := validating.NewChain(logger, vals...)

	cfg := validating.WebhookConfig{
		Name: "multiwebhook-deploymentValidator",
		Obj:  &extensionsv1beta1.Deployment{},
	}

	return validating.NewWebhook(cfg, valChain, ot, mrec, logger)
}
