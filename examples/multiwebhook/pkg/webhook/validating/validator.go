package validating

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// deploymentReplicasValidator will validate the replicas are between max and min (inclusive).
type deploymentReplicasValidator struct {
	maxReplicas int
	minReplicas int
	logger      kwhlog.Logger
}

func (d *deploymentReplicasValidator) Validate(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhvalidating.ValidatorResult, error) {
	depl, ok := obj.(*extensionsv1beta1.Deployment)
	if !ok {
		// If not a deployment just continue the validation chain(if there is one) and don't do nothing.
		return &kwhvalidating.ValidatorResult{Valid: true}, nil
	}

	reps := int(*depl.Spec.Replicas)
	if reps > d.maxReplicas {
		return &kwhvalidating.ValidatorResult{
			Valid:   false,
			Message: fmt.Sprintf("%d is not a valid replica number, deployment max replicas are %d", reps, d.maxReplicas),
		}, nil
	}

	if reps < d.minReplicas {
		return &kwhvalidating.ValidatorResult{
			Valid:   false,
			Message: fmt.Sprintf("%d is not a valid replica number, deployment min replicas are %d", reps, d.minReplicas),
		}, nil
	}

	return &kwhvalidating.ValidatorResult{Valid: true}, nil
}

type lantencyValidator struct {
	maxLatencyMS int
}

func (m *lantencyValidator) Validate(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhvalidating.ValidatorResult, error) {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	ms := time.Duration(rand.Intn(m.maxLatencyMS)) * time.Millisecond
	time.Sleep(ms)
	return &kwhvalidating.ValidatorResult{Valid: true}, nil
}
