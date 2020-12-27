package webhook

import (
	"context"
	"strings"
	"time"

	"github.com/slok/kubewebhook/v2/pkg/model"
)

// MeasureOpCommonData is the
type MeasureOpCommonData struct {
	WebhookID              string
	WebhookType            string
	AdmissionReviewVersion string
	Duration               time.Duration
	Success                bool
	ResourceName           string
	ResourceNamespace      string
	Operation              string
	ResourceKind           string
	DryRun                 bool
	WarningsNumber         int
}

// MeasureValidatingOpData is the data to measure webhook validating operation data.
type MeasureValidatingOpData struct {
	MeasureOpCommonData
	Allowed bool
}

// MeasureMutatingOpData is the data to measure webhook mutating operation data.
type MeasureMutatingOpData struct {
	MeasureOpCommonData
	Mutated bool
}

// MetricsRecorder knows how to record webhook recorder metrics.
type MetricsRecorder interface {
	MeasureValidatingWebhookReviewOp(ctx context.Context, data MeasureValidatingOpData)
	MeasureMutatingWebhookReviewOp(ctx context.Context, data MeasureMutatingOpData)
}

type noopMetricsRecorder int

// NoopMetricsRecorder is a no-op metrics recorder.
const NoopMetricsRecorder = noopMetricsRecorder(0)

var _ MetricsRecorder = NoopMetricsRecorder

func (noopMetricsRecorder) MeasureValidatingWebhookReviewOp(ctx context.Context, data MeasureValidatingOpData) {
}
func (noopMetricsRecorder) MeasureMutatingWebhookReviewOp(ctx context.Context, data MeasureMutatingOpData) {
}

type measuredWebhook struct {
	webhookID   string
	webhookKind model.WebhookKind
	rec         MetricsRecorder
	next        Webhook
}

// NewMeasuredWebhook returns a wrapped webhook that will measure the webhook operations.
func NewMeasuredWebhook(rec MetricsRecorder, next Webhook) Webhook {
	return measuredWebhook{
		webhookID:   next.ID(),
		webhookKind: next.Kind(),
		rec:         rec,
		next:        next,
	}
}

func (m measuredWebhook) ID() string              { return m.next.ID() }
func (m measuredWebhook) Kind() model.WebhookKind { return m.next.Kind() }
func (m measuredWebhook) Review(ctx context.Context, ar model.AdmissionReview) (resp model.AdmissionResponse, err error) {
	defer func(t0 time.Time) {
		gvk := ar.RequestGVK
		cData := MeasureOpCommonData{
			WebhookID:              m.webhookID,
			AdmissionReviewVersion: string(ar.Version),
			Duration:               time.Since(t0),
			Success:                err == nil,
			ResourceName:           ar.Name,
			ResourceNamespace:      ar.Namespace,
			Operation:              string(ar.Operation),
			ResourceKind:           strings.Trim(strings.Join([]string{gvk.Group, gvk.Version, gvk.Kind}, "/"), "/"),
			DryRun:                 ar.DryRun,
		}

		switch r := resp.(type) {
		case *model.ValidatingAdmissionResponse:
			cData.WebhookType = model.WebhookKindMutating
			cData.WarningsNumber = len(r.Warnings)
			m.rec.MeasureValidatingWebhookReviewOp(ctx, MeasureValidatingOpData{
				MeasureOpCommonData: cData,
				Allowed:             r.Allowed,
			})

		case *model.MutatingAdmissionResponse:
			cData.WebhookType = model.WebhookKindValidating
			cData.WarningsNumber = len(r.Warnings)
			m.rec.MeasureMutatingWebhookReviewOp(ctx, MeasureMutatingOpData{
				MeasureOpCommonData: cData,
				Mutated:             len(r.JSONPatchPatch) > 0 && string(r.JSONPatchPatch) != "[]",
			})

		default:
			// Unknown type, not measuring.
			// TODO(slok): Notify user ignore metrics.
		}

	}(time.Now())

	return m.next.Review(ctx, ar)
}
