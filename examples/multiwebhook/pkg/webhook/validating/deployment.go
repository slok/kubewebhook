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
	val := &deploymentReplicasValidator{
		maxReplicas: maxReplicas,
		minReplicas: minReplicas,
		logger:      logger,
	}

	cfg := validating.WebhookConfig{
		Name: "multiwebhook-deploymentValidator",
		Obj:  &extensionsv1beta1.Deployment{},
	}

	return validating.NewWebhook(cfg, val, ot, mrec, logger)
}
