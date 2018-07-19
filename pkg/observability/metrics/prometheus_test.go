package metrics_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"

	"github.com/slok/kubewebhook/pkg/observability/metrics"
)

func TestPrometheus(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		recordMetrics func(metrics.Recorder)
		expMetrics    []string
	}{
		{
			name: "Record admission review counts should set the correct metrics",
			recordMetrics: func(m metrics.Recorder) {
				m.IncAdmissionReview("testWH", "test", "v1/pods", admissionv1beta1.Create, metrics.ValidatingReviewKind)
				m.IncAdmissionReview("testWH", "test2", "v1/pods", admissionv1beta1.Create, metrics.MutatingReviewKind)
				m.IncAdmissionReview("testWH2", "test", "v1/ingress", admissionv1beta1.Update, metrics.ValidatingReviewKind)
				m.IncAdmissionReview("testWH", "test", "v1/pods", admissionv1beta1.Create, metrics.ValidatingReviewKind)
				m.IncAdmissionReviewError("testWH", "test", "v1/pods", admissionv1beta1.Create, metrics.ValidatingReviewKind)
				m.IncAdmissionReviewError("testWH", "test", "v1/pods", admissionv1beta1.Create, metrics.ValidatingReviewKind)
				m.IncAdmissionReviewError("testWH", "test", "v1/pods", admissionv1beta1.Create, metrics.ValidatingReviewKind)

			},
			expMetrics: []string{
				`kubewebhook_admission_webhook_admission_reviews_total{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH"} 2`,
				`kubewebhook_admission_webhook_admission_reviews_total{kind="validating",namespace="test",operation="UPDATE",resource="v1/ingress",webhook="testWH2"} 1`,
				`kubewebhook_admission_webhook_admission_reviews_total{kind="mutating",namespace="test2",operation="CREATE",resource="v1/pods",webhook="testWH"} 1`,
				`kubewebhook_admission_webhook_admission_review_errors_total{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH"} 3`,
			},
		},
		{
			name: "Record admission review duration should set the correct metrics",
			recordMetrics: func(m metrics.Recorder) {
				m.ObserveAdmissionReviewDuration("testWH", "test", "v1/pods", admissionv1beta1.Create, metrics.ValidatingReviewKind, now.Add(-1*time.Second))
				m.ObserveAdmissionReviewDuration("testWH", "test", "v1/pods", admissionv1beta1.Create, metrics.ValidatingReviewKind, now.Add(-2*time.Millisecond))
				m.ObserveAdmissionReviewDuration("testWH", "test", "v1/pods", admissionv1beta1.Create, metrics.ValidatingReviewKind, now.Add(-200*time.Millisecond))
				m.ObserveAdmissionReviewDuration("testWH", "test", "v1/pods", admissionv1beta1.Create, metrics.ValidatingReviewKind, now.Add(-20*time.Second))
			},
			expMetrics: []string{
				`kubewebhook_admission_webhook_admission_review_duration_seconds_bucket{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH",le="0.005"} 1`,
				`kubewebhook_admission_webhook_admission_review_duration_seconds_bucket{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH",le="0.01"} 1`,
				`kubewebhook_admission_webhook_admission_review_duration_seconds_bucket{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH",le="0.025"} 1`,
				`kubewebhook_admission_webhook_admission_review_duration_seconds_bucket{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH",le="0.05"} 1`,
				`kubewebhook_admission_webhook_admission_review_duration_seconds_bucket{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH",le="0.1"} 1`,
				`kubewebhook_admission_webhook_admission_review_duration_seconds_bucket{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH",le="0.25"} 2`,
				`kubewebhook_admission_webhook_admission_review_duration_seconds_bucket{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH",le="0.5"} 2`,
				`kubewebhook_admission_webhook_admission_review_duration_seconds_bucket{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH",le="1"} 2`,
				`kubewebhook_admission_webhook_admission_review_duration_seconds_bucket{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH",le="2.5"} 3`,
				`kubewebhook_admission_webhook_admission_review_duration_seconds_bucket{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH",le="5"} 3`,
				`kubewebhook_admission_webhook_admission_review_duration_seconds_bucket{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH",le="10"} 3`,
				`kubewebhook_admission_webhook_admission_review_duration_seconds_bucket{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH",le="+Inf"} 4`,
				`kubewebhook_admission_webhook_admission_review_duration_seconds_count{kind="validating",namespace="test",operation="CREATE",resource="v1/pods",webhook="testWH"} 4`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			reg := prometheus.NewRegistry()
			p := metrics.NewPrometheus(reg)

			test.recordMetrics(p)

			// Get the metrics handler and serve.
			h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/metrics", nil)
			h.ServeHTTP(rec, req)

			resp := rec.Result()

			// Check all metrics are present.
			if assert.Equal(http.StatusOK, resp.StatusCode) {
				body, _ := ioutil.ReadAll(resp.Body)
				for _, expMetric := range test.expMetrics {
					assert.Contains(string(body), expMetric, "metric not present on the result of metrics service")
				}
			}
		})
	}
}
