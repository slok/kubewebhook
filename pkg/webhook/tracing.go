package webhook

import (
	"context"
	"fmt"

	"github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/tracing"
)

type tracedWebhook struct {
	webhookID   string
	webhookKind model.WebhookKind
	tracer      tracing.Tracer
	next        Webhook
}

// NewTracedWebhook returns a wrapped webhook that will trace the webhook operations.
func NewTracedWebhook(tracer tracing.Tracer, next Webhook) Webhook {
	return tracedWebhook{
		webhookID:   next.ID(),
		webhookKind: next.Kind(),
		tracer:      tracer,
		next:        next,
	}
}

func (t tracedWebhook) ID() string              { return t.next.ID() }
func (t tracedWebhook) Kind() model.WebhookKind { return t.next.Kind() }
func (t tracedWebhook) Review(ctx context.Context, ar model.AdmissionReview) (resp model.AdmissionResponse, err error) {
	ctx = t.tracer.NewTrace(ctx, fmt.Sprintf("webhook.Review/%s", t.webhookID))
	t.tracer.AddTraceValues(ctx, map[string]interface{}{
		"webhook_id":                     t.webhookID,
		"admission_review_version":       ar.Version,
		"admission_review_user_uid":      ar.UserInfo.UID,
		"admission_review_user_username": ar.UserInfo.Username,
		"admission_review_user_groups":   ar.UserInfo.Groups,
		"admission_review_id":            ar.ID,
		"resource_name":                  ar.Name,
		"resource_namespace":             ar.Namespace,
		"operation":                      ar.Operation,
		"resource_kind":                  getResourceKind(ar),
		"dry_run":                        ar.DryRun,
	})

	defer func() {
		switch r := resp.(type) {
		case *model.ValidatingAdmissionResponse:
			t.tracer.AddTraceValues(ctx, map[string]interface{}{
				"webhook_type": model.WebhookKindMutating,
				"warnings":     r.Warnings,
				"has_warnings": len(r.Warnings) > 0,
				"allowed":      r.Allowed,
			})

		case *model.MutatingAdmissionResponse:
			t.tracer.AddTraceValues(ctx, map[string]interface{}{
				"webhook_type": model.WebhookKindValidating,
				"warnings":     r.Warnings,
				"has_warnings": len(r.Warnings) > 0,
				"mutated":      hasMutated(r),
			})

		default:
			// Unknown type, not traced.
			// TODO(slok): Notify user ignored traces.
		}

		t.tracer.EndTrace(ctx, err)
	}()

	return t.next.Review(ctx, ar)
}
