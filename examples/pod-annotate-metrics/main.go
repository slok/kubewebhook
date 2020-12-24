package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhprometheus "github.com/slok/kubewebhook/v2/pkg/metrics/prometheus"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhwebhook "github.com/slok/kubewebhook/v2/pkg/webhook"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
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
	logger := &kwhlog.Std{Debug: true}

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

	// Prepare metrics
	reg := prometheus.NewRegistry()
	metricsRec, err := kwhprometheus.NewRecorder(kwhprometheus.RecorderConfig{Registry: reg})
	if err != nil {
		return fmt.Errorf("could not create Prometheus metrics recorder: %w", err)
	}

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
	whHandler, err := kwhhttp.HandlerFor(kwhwebhook.NewMeasuredWebhook(metricsRec, wh))
	if err != nil {
		return fmt.Errorf("error creating webhook handler: %w", err)
	}

	errCh := make(chan error)
	// Serve webhook.
	go func() {
		logger.Infof("Listening on :8080")
		err = http.ListenAndServeTLS(":8080", cfg.certFile, cfg.keyFile, whHandler)
		if err != nil {
			errCh <- fmt.Errorf("error serving webhook: %w", err)
		}
		errCh <- nil
	}()

	// Serve metrics.
	go func() {
		logger.Infof("Listening metrics on :8081")
		err = http.ListenAndServe(":8081", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		if err != nil {
			errCh <- fmt.Errorf("error serving webhook metrics: %w", err)
		}
		errCh <- nil
	}()

	err = <-errCh
	if err != nil {
		return err
	}

	return nil
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
