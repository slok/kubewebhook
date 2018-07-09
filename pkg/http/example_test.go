package http_test

import (
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	whhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
	"github.com/slok/kubewebhook/pkg/webhook/validating"
)

// ExampleServeWebhook shows how to serve a validating webhook that denies all pods.
func ExampleServeWebhook() {
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
	wh, _ := validating.NewWebhook(v, &corev1.Pod{}, log.Dummy)

	// Get webhook handler and serve (webhooks need to be server with TLS).
	whHandler := whhttp.HandlerFor(wh)
	http.ListenAndServeTLS(":8080", "file.cert", "file.key", whHandler)
}

// ExampleServeMultipleWebhooks shows how to serve multiple webhooks in the same server.
func ExampleServeMultipleWebhooks() {
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
	vwh, _ := validating.NewWebhook(v, &corev1.Pod{}, log.Dummy)
	mwh, _ := mutating.NewStaticWebhook(m, &corev1.Pod{}, log.Dummy)

	// Create a muxer and handle different webhooks in different paths of the server.
	mux := http.NewServeMux()
	mux.Handle("/validate-pod", whhttp.HandlerFor(vwh))
	mux.Handle("/mutate-pod", whhttp.HandlerFor(mwh))
	http.ListenAndServeTLS(":8080", "file.cert", "file.key", mux)
}
