package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlogrus "github.com/slok/kubewebhook/v2/pkg/log/logrus"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhtracing "github.com/slok/kubewebhook/v2/pkg/tracing"
	kwhotel "github.com/slok/kubewebhook/v2/pkg/tracing/otel"
	kwhwebhook "github.com/slok/kubewebhook/v2/pkg/webhook"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	otelsdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type config struct {
	certFile string
	keyFile  string
}

func initFlags() *config {
	cfg := &config{}

	fl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fl.StringVar(&cfg.certFile, "tls-cert-file", "", "TLS certificate file")
	fl.StringVar(&cfg.keyFile, "tls-key-file", "", "TLS key file")

	fl.Parse(os.Args[1:])
	return cfg
}

func run() error {
	logrusLogEntry := logrus.NewEntry(logrus.New())
	logrusLogEntry.Logger.SetLevel(logrus.DebugLevel)
	logger := kwhlogrus.NewLogrus(logrusLogEntry)

	cfg := initFlags()

	// Create mutator.
	mt := kwhmutating.MutatorFunc(func(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return &kwhmutating.MutatorResult{}, nil
		}

		// Mutate our object with the required annotations.
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations["mutated"] = "true"
		pod.Annotations["mutator"] = "pod-annotate"

		return &kwhmutating.MutatorResult{
			MutatedObject: pod,
			Warnings:      []string{"pod mutated"},
		}, nil
	})

	// Prepare Tracer.
	tracer, stop, err := newTracer("pod-annotate-tracing")
	if err != nil {
		return fmt.Errorf("could not create tracer: %w", err)
	}
	defer stop()

	// Create webhook.
	mcfg := kwhmutating.WebhookConfig{
		ID:      "pod-annotate",
		Mutator: mt,
		Logger:  logger,
	}
	wh, err := kwhmutating.NewWebhook(mcfg)
	if err != nil {
		return fmt.Errorf("error creating webhook: %w", err)
	}

	// Get HTTP handler from webhook.
	whHandler, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{
		Webhook: kwhwebhook.NewTracedWebhook(tracer, wh),
		Tracer:  tracer,
		Logger:  logger,
	})
	if err != nil {
		return fmt.Errorf("error creating webhook handler: %w", err)
	}

	logger.Infof("Listening on :8080")
	return http.ListenAndServeTLS(":8080", cfg.certFile, cfg.keyFile, whHandler)
}

func newTracer(name string) (tracer kwhtracing.Tracer, stop func(), err error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, nil, err
	}

	tp := otelsdktrace.NewTracerProvider(
		otelsdktrace.WithBatcher(exporter),
		otelsdktrace.WithSampler(otelsdktrace.AlwaysSample()),
		otelsdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(name),
		)),
	)

	propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
	tracer = kwhotel.NewTracer(tp, propagator)
	stop = func() { _ = tp.Shutdown(context.Background()) }
	return tracer, stop, nil
}

func main() {
	err := run()
	if err != nil {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error running app: %s", err)
			os.Exit(1)
		}
	}
}
