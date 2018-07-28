package instrumenting_test

import (
	"context"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/mock"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mmetrics "github.com/slok/kubewebhook/mocks/observability/metrics"
	mwebhook "github.com/slok/kubewebhook/mocks/webhook"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	"github.com/slok/kubewebhook/pkg/webhook/internal/instrumenting"
)

func TestInstrumentedMetricsWebhook(t *testing.T) {
	tests := []struct {
		name   string
		aRev   *admissionv1beta1.AdmissionReview
		aResp  *admissionv1beta1.AdmissionResponse
		whName string
		whKind metrics.ReviewKind
		expErr bool
	}{
		{
			name: "A regular revision should add the happy path metrics without error",
			aRev: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID: "test",
				},
			},
			aResp:  &admissionv1beta1.AdmissionResponse{},
			whName: "test-webhook",
			whKind: metrics.ValidatingReviewKind,
		},
		{
			name: "A revision with error should add the path metrics with error",
			aRev: &admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID: "test",
				},
			},
			aResp: &admissionv1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Status: metav1.StatusFailure,
				},
			},
			whName: "test-error-webhook",
			whKind: metrics.MutatingReviewKind,
			expErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// Mocks
			mwh := &mwebhook.Webhook{}
			mwh.On("Review", mock.Anything, mock.Anything).Once().Return(test.aResp)

			mm := &mmetrics.Recorder{}
			mm.On("IncAdmissionReview", test.whName, mock.Anything, mock.Anything, mock.Anything, test.whKind).Once()
			mm.On("ObserveAdmissionReviewDuration", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once()
			if test.expErr {
				mm.On("IncAdmissionReviewError", test.whName, mock.Anything, mock.Anything, mock.Anything, test.whKind).Once()
			}

			wh := instrumenting.Webhook{
				Webhook:         mwh,
				WebhookName:     test.whName,
				ReviewKind:      test.whKind,
				MetricsRecorder: mm,
				Tracer:          &opentracing.NoopTracer{},
			}

			wh.Review(context.TODO(), test.aRev)

			// Check calls.
			mwh.AssertExpectations(t)
			mm.AssertExpectations(t)
		})
	}
}
