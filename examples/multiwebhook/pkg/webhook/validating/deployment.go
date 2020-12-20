package validating

import (
	"github.com/slok/kubewebhook/v2/pkg/log"
	"github.com/slok/kubewebhook/v2/pkg/webhook"
	"github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
)

// NewDeploymentWebhook returns a new deployment validationg webhook.
func NewDeploymentWebhook(minReplicas, maxReplicas int, logger log.Logger) (webhook.Webhook, error) {

	// Create validators.
	repVal := &deploymentReplicasValidator{
		maxReplicas: maxReplicas,
		minReplicas: minReplicas,
		logger:      logger,
	}

	vals := []validating.Validator{
		&lantencyValidator{maxLatencyMS: 20},
		&lantencyValidator{maxLatencyMS: 120},
		&lantencyValidator{maxLatencyMS: 300},
		&lantencyValidator{maxLatencyMS: 10},
		&lantencyValidator{maxLatencyMS: 175},
		&lantencyValidator{maxLatencyMS: 80},
		&lantencyValidator{maxLatencyMS: 10},
		repVal,
	}

	return validating.NewWebhook(
		validating.WebhookConfig{
			ID:        "multiwebhook-deploymentValidator",
			Obj:       &extensionsv1beta1.Deployment{},
			Validator: validating.NewChain(logger, vals...),
			Logger:    logger,
		})
}
