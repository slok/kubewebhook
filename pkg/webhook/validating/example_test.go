package validating_test

import (
	"context"
	"fmt"
	"regexp"

	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/webhook/validating"
)

// ExampleIngressHostValidatingWebhook shows how you would create a ingress validating webhook that checks
// if an ingress has any rule with an invalid host that doesn't match the valid host regex and if is invalid
// will not accept the ingress.
func ExampleIngressHostValidatingWebhook() {
	// Create the regex to validate the hosts.
	validHost := regexp.MustCompile(`^.*\.batman\.best\.superhero\.io$`)

	// Create our validator that will check the host on each rule of the received ingress to
	// allow or disallow the ingress.
	ivh := validating.ValidatorFunc(func(_ context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
		ingress, ok := obj.(*extensionsv1beta1.Ingress)

		if !ok {
			return false, validating.ValidatorResult{}, fmt.Errorf("not an ingress")
		}

		for _, r := range ingress.Spec.Rules {
			if !validHost.MatchString(r.Host) {
				res := validating.ValidatorResult{
					Valid:   false,
					Message: fmt.Sprintf("%s ingress host doesn't match %s regex", r.Host, validHost),
				}
				return false, res, nil
			}
		}

		res := validating.ValidatorResult{
			Valid:   true,
			Message: "all hosts in the ingress are valid",
		}
		return false, res, nil
	})

	// Create webhook (usage of webhook not in this example).
	validating.NewWebhook(ivh, &extensionsv1beta1.Ingress{}, log.Dummy)
}
