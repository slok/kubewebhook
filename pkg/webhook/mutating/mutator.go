package mutating

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/v2/pkg/log"
	"github.com/slok/kubewebhook/v2/pkg/model"
)

// MutatorResult is the result of a mutator.
type MutatorResult struct {
	// StopChain will stop the chain of validators in case there is a chain set.
	StopChain bool
	// MutatedObject is the object that has been mutated. If is nil, it will be used the one
	// received by the Mutator.
	MutatedObject metav1.Object
	// Warnings are special messages that can be set to warn the user (e.g deprecation messages, almost invalid resources...).
	Warnings []string
}

// Mutator knows how to mutate the received kubernetes object.
type Mutator interface {
	// Mutate receives a Kubernetes resource object to be mutated, it must
	// return an error or a mutation result. What the mutator returns
	// as result.MutatedObject is the object that will be used as the mutation.
	// It must be of the same type of the received one (if is a Pod, it must return a Pod)
	// if no object is returned, it will be used the received one as the mutated one.
	// Also receives the webhook admission review in case it wants more context and
	// information of the review.
	// Mutators can be grouped in chains, that's why we have a `StopChain` boolean
	// in the result, to stop executing the validators chain.
	Mutate(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (result *MutatorResult, err error)
}

//go:generate mockery --case underscore --output mutatingmock --outpkg mutatingmock --name Mutator

// MutatorFunc is a helper type to create mutators from functions.
type MutatorFunc func(context.Context, *model.AdmissionReview, metav1.Object) (*MutatorResult, error)

// Mutate satisfies Mutator interface.
func (f MutatorFunc) Mutate(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*MutatorResult, error) {
	return f(ctx, ar, obj)
}

// Chain is a chain of mutators that will execute secuentially all the
// mutators that have been added to it. It satisfies Mutator interface.
type Chain struct {
	mutators []Mutator
	logger   log.Logger
}

// NewChain returns a new chain.
func NewChain(logger log.Logger, mutators ...Mutator) *Chain {
	return &Chain{
		mutators: mutators,
		logger:   logger,
	}
}

// Mutate will execute all the mutation chain.
func (c *Chain) Mutate(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*MutatorResult, error) {
	var warnings []string
	for _, mt := range c.mutators {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("mutator chain not finished correctly, context done")
		default:
			res, err := mt.Mutate(ctx, ar, obj)
			if err != nil {
				return nil, err
			}

			if res == nil {
				return nil, fmt.Errorf("validator result can't be `nil`")
			}

			// Don't lose the data through the chain, set warnings and pass around the mutated object.
			warnings = append(warnings, res.Warnings...)
			if res.MutatedObject != nil {
				obj = res.MutatedObject
			}

			if res.StopChain {
				res.Warnings = warnings
				return res, nil
			}
		}
	}

	return &MutatorResult{
		MutatedObject: obj,
		Warnings:      warnings,
	}, nil
}
