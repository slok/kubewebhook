package metrics_test

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	whhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
)

// Prometheus shows how to serve a webhook and its prometheus metrics in a separate server.
func ExamplePrometheus_servePrometheusMetrics() {

	// Create the prometheus registry. This registry can be used for custom metrics
	// when serving the prometheus metrics our custom metrics and the webhook metrics
	// will be served.
	reg := prometheus.NewRegistry()

	// Create our metrics service.
	metricsRec := metrics.NewPrometheus(reg)

	// Create a stub mutator.
	m := mutating.MutatorFunc(func(_ context.Context, obj metav1.Object) (bool, error) {
		return false, nil
	})

	// Create webhooks (don't check error).
	mcfg := mutating.WebhookConfig{
		Name: "instrucmentedWebhook",
		Obj:  &corev1.Pod{},
	}
	mwh, _ := mutating.NewWebhook(mcfg, m, nil, metricsRec, nil)

	// Run our webhook server (not checking error in this example).
	whHandler, _ := whhttp.HandlerFor(mwh)
	go http.ListenAndServeTLS(":8080", "file.cert", "file.key", whHandler)

	// Run our metrics in a separate port (not checking error in this example).
	promHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	go http.ListenAndServe(":8081", promHandler)
}
