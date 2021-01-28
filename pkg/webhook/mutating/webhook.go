package mutating

import (
	"context"
	"encoding/json"
	"fmt"

	"gomodules.xyz/jsonpatch/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/v2/pkg/log"
	"github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/webhook"
	"github.com/slok/kubewebhook/v2/pkg/webhook/internal/helpers"
)

// WebhookConfig is the Mutating webhook configuration.
type WebhookConfig struct {
	// ID is the id of the webhook.
	ID string
	// Object is the object of the webhook, to use multiple types on the same webhook or
	// type inference, don't set this field (will be `nil`).
	Obj metav1.Object
	// Mutator is the webhook mutator.
	Mutator Mutator
	// Logger is the app logger.
	Logger log.Logger
}

func (c *WebhookConfig) defaults() error {
	if c.ID == "" {
		return fmt.Errorf("id is required")
	}

	if c.Mutator == nil {
		return fmt.Errorf("mutator is required")
	}

	if c.Logger == nil {
		c.Logger = log.Noop
	}
	c.Logger = c.Logger.WithValues(log.Kv{"webhook-id": c.ID, "webhhok-type": "mutating"})

	return nil
}

type mutatingWebhook struct {
	id            string
	objectCreator helpers.ObjectCreator
	mutator       Mutator
	cfg           WebhookConfig
	logger        log.Logger
}

// NewWebhook is a mutating webhook and will return a webhook ready for a type of resource.
// It will mutate the received resources.
// This webhook will always allow the admission of the resource, only will deny in case of error.
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

	return &mutatingWebhook{
		objectCreator: oc,
		id:            cfg.ID,
		mutator:       cfg.Mutator,
		cfg:           cfg,
		logger:        cfg.Logger,
	}, nil
}

func (w mutatingWebhook) ID() string { return w.id }

func (w mutatingWebhook) Kind() model.WebhookKind { return model.WebhookKindMutating }

func (w mutatingWebhook) Review(ctx context.Context, ar model.AdmissionReview) (model.AdmissionResponse, error) {
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

	mutatingObj, ok := runtimeObj.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("impossible to type assert the deep copy to metav1.Object")
	}

	res, err := w.mutatingAdmissionReview(ctx, ar, raw, mutatingObj)
	if err != nil {
		return nil, err
	}

	w.logger.WithCtxValues(ctx).Debugf("Webhook mutating review finished with: '%s' JSON Patch", string(res.JSONPatchPatch))

	return res, nil
}

func (w mutatingWebhook) mutatingAdmissionReview(ctx context.Context, ar model.AdmissionReview, rawObj []byte, objForMutation metav1.Object) (*model.MutatingAdmissionResponse, error) {
	// Mutate the object.
	res, err := w.mutator.Mutate(ctx, &ar, objForMutation)
	if err != nil {
		return nil, fmt.Errorf("could not mutate object: %w", err)
	}

	if res == nil {
		return nil, fmt.Errorf("result is required, mutator result is nil")
	}

	// If the user returned a mutated object, it will not be used the one we provided to the mutator.
	// if nil then, we use the one we provided.
	mutatedObj := objForMutation
	if res.MutatedObject != nil {
		mutatedObj = res.MutatedObject
	}
	mutatedJSON, err := json.Marshal(mutatedObj)
	if err != nil {
		return nil, fmt.Errorf("could not marshal into JSON mutated object: %w", err)
	}

	patch, err := jsonpatch.CreatePatch(rawObj, mutatedJSON)
	if err != nil {
		return nil, fmt.Errorf("could not create JSON patch: %w", err)
	}

	marshalledPatch, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("could not mashal into JSON, the JSON patch: %w", err)
	}

	// Forge response.
	return &model.MutatingAdmissionResponse{
		ID:             ar.ID,
		JSONPatchPatch: marshalledPatch,
		Warnings:       res.Warnings,
	}, nil
}
