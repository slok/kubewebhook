package http_test

import (
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	whhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
	"github.com/slok/kubewebhook/pkg/webhook/validating"
)

// ServeWebhook shows how to serve a validating webhook that denies all pods.
func ExampleHandlerFor_serveWebhook() {
	// Create (in)validator.
	v := validating.ValidatorFunc(func(_ context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
		// Assume always is a pod (you should check type assertion is ok to not panic).
		pod := obj.(*corev1.Pod)

		res := validating.ValidatorResult{
			Valid:   false,
			Message: fmt.Sprintf("%s/%s denied because sll pods will be denied", pod.Namespace, pod.Name),
		}
		return false, res, nil
	})

	// Create webhook (don't check error).
	cfg := validating.WebhookConfig{
		Name: "serveWebhook",
		Obj:  &corev1.Pod{},
	}
	wh, _ := validating.NewWebhook(cfg, v, nil, nil, nil)

	// Get webhook handler and serve (webhooks need to be server with TLS).
	whHandler, _ := whhttp.HandlerFor(wh)
	http.ListenAndServeTLS(":8080", "file.cert", "file.key", whHandler)
}

// ServeMultipleWebhooks shows how to serve multiple webhooks in the same server.
func ExampleHandlerFor_serveMultipleWebhooks() {
	// Create (in)validator.
	v := validating.ValidatorFunc(func(_ context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
		// Assume always is a pod (you should check type assertion is ok to not panic).
		pod := obj.(*corev1.Pod)

		res := validating.ValidatorResult{
			Valid:   false,
			Message: fmt.Sprintf("%s/%s denied because sll pods will be denied", pod.Namespace, pod.Name),
		}
		return false, res, nil
	})

	// Create a stub mutator.
	m := mutating.MutatorFunc(func(_ context.Context, obj metav1.Object) (bool, error) {
		return false, nil
	})

	// Create webhooks (don't check error).
	vcfg := validating.WebhookConfig{
		Name: "validatingServeWebhook",
		Obj:  &corev1.Pod{},
	}
	vwh, _ := validating.NewWebhook(vcfg, v, nil, nil, nil)
	vwhHandler, _ := whhttp.HandlerFor(vwh)

	mcfg := mutating.WebhookConfig{
		Name: "muratingServeWebhook",
		Obj:  &corev1.Pod{},
	}
	mwh, _ := mutating.NewWebhook(mcfg, m, nil, nil, nil)
	mwhHandler, _ := whhttp.HandlerFor(mwh)

	// Create a muxer and handle different webhooks in different paths of the server.
	mux := http.NewServeMux()
	mux.Handle("/validate-pod", vwhHandler)
	mux.Handle("/mutate-pod", mwhHandler)
	http.ListenAndServeTLS(":8080", "file.cert", "file.key", mux)
}
