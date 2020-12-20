package http_test

import (
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	whhttp "github.com/slok/kubewebhook/v2/pkg/http"
	"github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	"github.com/slok/kubewebhook/v2/pkg/webhook/validating"
)

// ServeWebhook shows how to serve a validating webhook that denies all pods.
func ExampleHandlerFor_serveWebhook() {
	// Create (in)validator.
	v := validating.ValidatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return &validating.ValidatorResult{Valid: true}, nil
		}

		return &validating.ValidatorResult{
			Valid:   false,
			Message: fmt.Sprintf("%s/%s denied because all pods will be denied", pod.Namespace, pod.Name),
		}, nil
	})

	// Create webhook (don't check error).
	cfg := validating.WebhookConfig{
		ID:        "serveWebhook",
		Obj:       &corev1.Pod{},
		Validator: v,
	}
	wh, _ := validating.NewWebhook(cfg)

	// Get webhook handler and serve (webhooks need to be server with TLS).
	whHandler, _ := whhttp.HandlerFor(wh)
	_ = http.ListenAndServeTLS(":8080", "file.cert", "file.key", whHandler)
}

// ServeMultipleWebhooks shows how to serve multiple webhooks in the same server.
func ExampleHandlerFor_serveMultipleWebhooks() {
	// Create (in)validator.
	v := validating.ValidatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
		// Assume always is a pod (you should check type assertion is ok to not panic).
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return &validating.ValidatorResult{Valid: true}, nil
		}

		return &validating.ValidatorResult{
			Valid:   false,
			Message: fmt.Sprintf("%s/%s denied because all pods will be denied", pod.Namespace, pod.Name),
		}, nil
	})

	// Create a stub mutator.
	m := mutating.MutatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*mutating.MutatorResult, error) {
		return &mutating.MutatorResult{}, nil
	})

	// Create webhooks (don't check error).
	vcfg := validating.WebhookConfig{
		ID:        "validatingServeWebhook",
		Obj:       &corev1.Pod{},
		Validator: v,
	}
	vwh, _ := validating.NewWebhook(vcfg)
	vwhHandler, _ := whhttp.HandlerFor(vwh)

	mcfg := mutating.WebhookConfig{
		ID:      "muratingServeWebhook",
		Obj:     &corev1.Pod{},
		Mutator: m,
	}
	mwh, _ := mutating.NewWebhook(mcfg)
	mwhHandler, _ := whhttp.HandlerFor(mwh)

	// Create a muxer and handle different webhooks in different paths of the server.
	mux := http.NewServeMux()
	mux.Handle("/validate-pod", vwhHandler)
	mux.Handle("/mutate-pod", mwhHandler)
	_ = http.ListenAndServeTLS(":8080", "file.cert", "file.key", mux)
}
