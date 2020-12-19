package validating

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/model"
	"github.com/slok/kubewebhook/pkg/webhook/validating"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// deploymentReplicasValidator will validate the replicas are between max and min (inclusive).
type deploymentReplicasValidator struct {
	maxReplicas int
	minReplicas int
	logger      log.Logger
}

func (d *deploymentReplicasValidator) Validate(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
	depl, ok := obj.(*extensionsv1beta1.Deployment)
	if !ok {
		// If not a deployment just continue the validation chain(if there is one) and don't do nothing.
		return &validating.ValidatorResult{Valid: true}, nil
	}

	reps := int(*depl.Spec.Replicas)
	if reps > d.maxReplicas {
		return &validating.ValidatorResult{
			Valid:   false,
			Message: fmt.Sprintf("%d is not a valid replica number, deployment max replicas are %d", reps, d.maxReplicas),
		}, nil
	}

	if reps < d.minReplicas {
		return &validating.ValidatorResult{
			Valid:   false,
			Message: fmt.Sprintf("%d is not a valid replica number, deployment min replicas are %d", reps, d.minReplicas),
		}, nil
	}

	return &validating.ValidatorResult{Valid: true}, nil
}

type lantencyValidator struct {
	maxLatencyMS int
}

func (m *lantencyValidator) Validate(_ context.Context, _ *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	ms := time.Duration(rand.Intn(m.maxLatencyMS)) * time.Millisecond
	time.Sleep(ms)
	return &validating.ValidatorResult{Valid: true}, nil
}
