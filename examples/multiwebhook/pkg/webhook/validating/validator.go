package validating

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/slok/kubewebhook/pkg/log"
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

func (d *deploymentReplicasValidator) Validate(_ context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
	depl, ok := obj.(*extensionsv1beta1.Deployment)
	if !ok {
		// If not a deployment just continue the validation chain(if there is one) and don't do nothing.
		return false, validating.ValidatorResult{Valid: true}, nil
	}

	// Mutate our object with the required annotations.
	reps := int(*depl.Spec.Replicas)
	if reps > d.maxReplicas {
		return true, validating.ValidatorResult{
			Valid:   false,
			Message: fmt.Sprintf("%d is not a valid replica number, deployment max replicas are %d", reps, d.maxReplicas),
		}, nil
	}

	if reps < d.minReplicas {
		return true, validating.ValidatorResult{
			Valid:   false,
			Message: fmt.Sprintf("%d is not a valid replica number, deployment min replicas are %d", reps, d.minReplicas),
		}, nil
	}

	return false, validating.ValidatorResult{Valid: true}, nil
}

type lantencyValidator struct {
	maxLatencyMS int
}

func (m *lantencyValidator) Validate(_ context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	ms := time.Duration(rand.Intn(m.maxLatencyMS)) * time.Millisecond
	time.Sleep(ms)
	return false, validating.ValidatorResult{Valid: true}, nil
}
