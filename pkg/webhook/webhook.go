package webhook

import (
	"context"

	"github.com/slok/kubewebhook/pkg/model"
)

// Kind is the webhook kind.
type Kind string

const (
	// KindMutating is the kind of the webhooks that mutate.
	KindMutating = "mutating"
	// KindValidating is the kind of the webhooks that validate.
	KindValidating = "validating"
)

// Webhook knows how to handle the admission reviews, in other words Webhook is a dynamic
// admission webhook for Kubernetes.
type Webhook interface {
	// The id of the webhook.
	ID() string
	// The kind of the webhook.
	Kind() Kind
	// Review will handle the admission review and return the AdmissionResponse with the result of the admission
	// error, mutation...
	Review(ctx context.Context, ar model.AdmissionReview) (model.AdmissionResponse, error)
}

//go:generate mockery --case underscore --output webhookmock --outpkg webhookmock --name Webhook
