package prometheus_test

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metrics "github.com/slok/kubewebhook/pkg/metrics/prometheus"
	"github.com/slok/kubewebhook/pkg/webhook"
)

func getCommonData() webhook.MeasureOpCommonData {
	return webhook.MeasureOpCommonData{
		WebhookID:              "test-wh",
		WebhookType:            "validation",
		AdmissionReviewVersion: "v1",
		Duration:               42 * time.Millisecond,
		Success:                false,
		ResourceName:           "test",
		ResourceNamespace:      "test-ns",
		Operation:              "delete",
		ResourceKind:           "core/v1/Pod",
		DryRun:                 true,
		WarningsNumber:         5,
	}
}

func TestRecorder(t *testing.T) {
	tests := map[string]struct {
		config     metrics.RecorderConfig
		measure    func(r *metrics.Recorder)
		expMetrics []string
	}{
		"Measure validation webhook review.": {
			measure: func(r *metrics.Recorder) {
				c1 := getCommonData()
				c2 := getCommonData()
				c2.WebhookID = "test2-wh"
				c2.Duration = 3 * time.Second
				c2.Operation = "update"
				c2.WarningsNumber = 2
				r.MeasureValidatingWebhookReviewOp(context.TODO(), webhook.MeasureValidatingOpData{MeasureOpCommonData: c1, Allowed: true})
				r.MeasureValidatingWebhookReviewOp(context.TODO(), webhook.MeasureValidatingOpData{MeasureOpCommonData: c2, Allowed: false})
			},
			expMetrics: []string{
				`# HELP kubewebhook_validating_webhook_review_duration_seconds The duration of the admission review handled by a validating webhook.`,
				`# TYPE kubewebhook_validating_webhook_review_duration_seconds histogram`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1",le="0.005"} 0`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1",le="0.01"} 0`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1",le="0.025"} 0`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1",le="0.05"} 0`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1",le="0.1"} 0`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1",le="0.25"} 0`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1",le="0.5"} 0`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1",le="1"} 0`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1",le="2.5"} 0`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1",le="5"} 1`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1",le="10"} 1`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1",le="+Inf"} 1`,
				`kubewebhook_validating_webhook_review_duration_seconds_count{allowed="false",dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1"} 1`,

				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.005"} 0`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.01"} 0`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.025"} 0`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.05"} 1`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.1"} 1`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.25"} 1`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.5"} 1`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="1"} 1`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="2.5"} 1`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="5"} 1`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="10"} 1`,
				`kubewebhook_validating_webhook_review_duration_seconds_bucket{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="+Inf"} 1`,
				`kubewebhook_validating_webhook_review_duration_seconds_count{allowed="true",dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1"} 1`,

				`# HELP kubewebhook_webhook_review_warnings_total The total number warnings the webhooks are returning on the review process.`,
				`# TYPE kubewebhook_webhook_review_warnings_total counter`,
				`kubewebhook_webhook_review_warnings_total{dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1"} 5`,
				`kubewebhook_webhook_review_warnings_total{dry_run="true",operation="update",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test2-wh",webhook_version="v1"} 2`,
			},
		},

		"Measure mutating webhook review.": {
			measure: func(r *metrics.Recorder) {
				c1 := getCommonData()
				c2 := getCommonData()
				c2.WebhookID = "test42-wh"
				c2.Duration = 1300 * time.Millisecond
				c2.Operation = "create"
				c2.WarningsNumber = 10
				r.MeasureMutatingWebhookReviewOp(context.TODO(), webhook.MeasureMutatingOpData{MeasureOpCommonData: c1, Mutated: true})
				r.MeasureMutatingWebhookReviewOp(context.TODO(), webhook.MeasureMutatingOpData{MeasureOpCommonData: c2, Mutated: false})
			},
			expMetrics: []string{
				`# HELP kubewebhook_mutating_webhook_review_duration_seconds The duration of the admission review handled by a mutating webhook.`,
				`# TYPE kubewebhook_mutating_webhook_review_duration_seconds histogram`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1",le="0.005"} 0`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1",le="0.01"} 0`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1",le="0.025"} 0`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1",le="0.05"} 0`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1",le="0.1"} 0`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1",le="0.25"} 0`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1",le="0.5"} 0`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1",le="1"} 0`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1",le="2.5"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1",le="5"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1",le="10"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1",le="+Inf"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_count{dry_run="true",mutated="false",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1"} 1`,

				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.005"} 0`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.01"} 0`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.025"} 0`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.05"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.1"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.25"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="0.5"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="1"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="2.5"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="5"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="10"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_bucket{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1",le="+Inf"} 1`,
				`kubewebhook_mutating_webhook_review_duration_seconds_count{dry_run="true",mutated="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1"} 1`,

				`# HELP kubewebhook_webhook_review_warnings_total The total number warnings the webhooks are returning on the review process.`,
				`# TYPE kubewebhook_webhook_review_warnings_total counter`,
				`kubewebhook_webhook_review_warnings_total{dry_run="true",operation="create",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test42-wh",webhook_version="v1"} 10`,
				`kubewebhook_webhook_review_warnings_total{dry_run="true",operation="delete",resource_kind="core/v1/Pod",resource_namespace="test-ns",success="false",webhook_id="test-wh",webhook_version="v1"} 5`,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			reg := prometheus.NewRegistry()
			test.config.Registry = reg
			rec, err := metrics.NewRecorder(test.config)
			require.NoError(err)

			test.measure(rec)

			h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
			allMetrics, err := ioutil.ReadAll(w.Result().Body)
			require.NoError(err)

			// Check metrics.
			for _, expMetric := range test.expMetrics {
				assert.Contains(string(allMetrics), expMetric)
			}
		})
	}
}
