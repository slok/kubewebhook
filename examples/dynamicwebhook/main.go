package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	mutatingwh "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
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

func main() {
	logger := &kwhlog.Std{Debug: true}

	cfg := initFlags()

	// Create our mutator.
	mt := mutatingwh.MutatorFunc(func(_ context.Context, ar *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
		labels := obj.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		labels[fmt.Sprintf("kubewebhook-%s", ar.Version)] = "mutated"
		obj.SetLabels(labels)

		return &kwhmutating.MutatorResult{MutatedObject: obj}, nil
	})

	// We don't use any type, it works for any type.
	mcfg := mutatingwh.WebhookConfig{
		ID:      "labeler",
		Mutator: mt,
		Logger:  logger,
	}
	wh, err := mutatingwh.NewWebhook(mcfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook: %s", err)
		os.Exit(1)
	}

	// Get the handler for our webhook.
	whHandler, err := kwhhttp.HandlerFor(wh)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook handler: %s", err)
		os.Exit(1)
	}
	logger.Infof("Listening on :8080")
	err = http.ListenAndServeTLS(":8080", cfg.certFile, cfg.keyFile, whHandler)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error serving webhook: %s", err)
		os.Exit(1)
	}
}
