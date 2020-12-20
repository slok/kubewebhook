package webhook

import (
	"context"

	"github.com/slok/kubewebhook/v2/pkg/model"
)

// Webhook knows how to handle the admission reviews, in other words Webhook is a dynamic
// admission webhook for Kubernetes.
type Webhook interface {
	// The id of the webhook.
	ID() string
	// The kind of the webhook.
	Kind() model.WebhookKind
	// Review will handle the admission review and return the AdmissionResponse with the result of the admission
	// error, mutation...
	Review(ctx context.Context, ar model.AdmissionReview) (model.AdmissionResponse, error)
}

//go:generate mockery --case underscore --output webhookmock --outpkg webhookmock --name Webhook
