package validating

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/v2/pkg/log"
	"github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/webhook"
	"github.com/slok/kubewebhook/v2/pkg/webhook/internal/helpers"
)

// WebhookConfig is the Validating webhook configuration.
type WebhookConfig struct {
	// ID is the id of the webhook.
	ID string
	// Object is the object of the webhook, to use multiple types on the same webhook or
	// type inference, don't set this field (will be `nil`).
	Obj metav1.Object
	// Validator is the webhook validator.
	Validator Validator
	// Logger is the app logger.
	Logger log.Logger
}

func (c *WebhookConfig) defaults() error {
	if c.ID == "" {
		return fmt.Errorf("id is required")
	}

	if c.Validator == nil {
		return fmt.Errorf("validator is required")
	}

	if c.Logger == nil {
		c.Logger = log.Noop
	}
	c.Logger = c.Logger.WithValues(log.Kv{"webhook-id": c.ID, "webhook-type": "validating"})

	return nil
}

// NewWebhook is a validating webhook and will return a webhook ready for a type of resource
// it will validate the received resources.
func NewWebhook(cfg WebhookConfig) (webhook.Webhook, error) {
	if err := cfg.defaults(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// If we don't have the type of the object create a dynamic object creator that will
	// infer the type.
	var oc helpers.ObjectCreator
	if cfg.Obj != nil {
		oc = helpers.NewStaticObjectCreator(cfg.Obj)
	} else {
		oc = helpers.NewDynamicObjectCreator()
	}

	// Create our webhook and wrap for instrumentation (metrics and tracing).
	return &validatingWebhook{
		id:            cfg.ID,
		objectCreator: oc,
		validator:     cfg.Validator,
		cfg:           cfg,
		logger:        cfg.Logger,
	}, nil
}

type validatingWebhook struct {
	id            string
	objectCreator helpers.ObjectCreator
	validator     Validator
	cfg           WebhookConfig
	logger        log.Logger
}

func (w validatingWebhook) ID() string { return w.id }

func (w validatingWebhook) Kind() model.WebhookKind { return model.WebhookKindValidating }

func (w validatingWebhook) Review(ctx context.Context, ar model.AdmissionReview) (model.AdmissionResponse, error) {
	// Delete operations don't have body because should be gone on the deletion, instead they have the body
	// of the object we want to delete as an old object.
	raw := ar.NewObjectRaw
	if ar.Operation == model.OperationDelete {
		raw = ar.OldObjectRaw
	}

	// Create a new object from the raw type.
	runtimeObj, err := w.objectCreator.NewObject(raw)
	if err != nil {
		return nil, fmt.Errorf("could not create object from raw: %w", err)
	}

	validatingObj, ok := runtimeObj.(metav1.Object)
	// Get the object.
	if !ok {
		return nil, fmt.Errorf("impossible to type assert the deep copy to metav1.Object")
	}

	res, err := w.validator.Validate(ctx, &ar, validatingObj)
	if err != nil {
		return nil, fmt.Errorf("validator error: %w", err)
	}

	if res == nil {
		return nil, fmt.Errorf("result is required, validator result is nil")
	}

	w.logger.WithCtxValues(ctx).WithValues(log.Kv{"valid": res.Valid}).Debugf("Webhook validating review finished with '%t' result", res.Valid)

	// Forge response.
	return &model.ValidatingAdmissionResponse{
		ID:       ar.ID,
		Allowed:  res.Valid,
		Message:  res.Message,
		Warnings: res.Warnings,
	}, nil
}
