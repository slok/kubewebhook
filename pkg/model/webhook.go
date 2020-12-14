package model

// WebhookKind is the webhook kind.
type WebhookKind string

// WebhookVersion is the webhook version.
type WebhookVersion string

const (
	// WebhookKindMutating is the kind of the webhooks that mutate.
	WebhookKindMutating = "mutating"
	// WebhookKindValidating is the kind of the webhooks that validate.
	WebhookKindValidating = "validating"
)
