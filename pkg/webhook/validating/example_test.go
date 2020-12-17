package validating_test

import (
	"context"
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/model"
	"github.com/slok/kubewebhook/pkg/webhook/validating"
)

// IngressHostValidatingWebhook shows how you would create a ingress validating webhook that checks
// if an ingress has any rule with an invalid host that doesn't match the valid host regex and if is invalid
// will not accept the ingress.
func ExampleValidator_ingressHostValidatingWebhook() {
	// Create the regex to validate the hosts.
	validHost := regexp.MustCompile(`^.*\.batman\.best\.superhero\.io$`)

	// Create our validator that will check the host on each rule of the received ingress to
	// allow or disallow the ingress.
	ivh := validating.ValidatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
		ingress, ok := obj.(*extensionsv1beta1.Ingress)
		if !ok {
			return &validating.ValidatorResult{Valid: true}, fmt.Errorf("not an ingress")
		}

		for _, r := range ingress.Spec.Rules {
			if !validHost.MatchString(r.Host) {
				return &validating.ValidatorResult{
					Valid:   false,
					Message: fmt.Sprintf("%s ingress host doesn't match %s regex", r.Host, validHost),
				}, nil
			}
		}

		return &validating.ValidatorResult{
			Valid:   true,
			Message: "all hosts in the ingress are valid",
		}, nil
	})

	// Create webhook (usage of webhook not in this example).
	_, _ = validating.NewWebhook(validating.WebhookConfig{
		ID:        "example",
		Obj:       &extensionsv1beta1.Ingress{},
		Validator: ivh,
	})
}

// chainValidatingWebhook shows how you would create a validating chain.
func ExampleValidator_chainValidatingWebhook() {
	fakeVal := validating.ValidatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
		return &validating.ValidatorResult{Valid: true}, nil
	})

	fakeVal2 := validating.ValidatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
		return &validating.ValidatorResult{Valid: true}, nil
	})

	fakeVal3 := validating.ValidatorFunc(func(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
		return &validating.ValidatorResult{Valid: true}, nil
	})

	// Create our webhook using a validator chain.
	_, _ = validating.NewWebhook(validating.WebhookConfig{
		ID:        "podWebhook",
		Obj:       &corev1.Pod{},
		Validator: validating.NewChain(log.Dummy, fakeVal, fakeVal2, fakeVal3),
	})

}
