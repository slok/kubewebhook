package validating

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/pkg/log"
)

// ValidatorResult is the result of a validator.
type ValidatorResult struct {
	Valid   bool
	Message string
}

// Validator knows how to validate the received kubernetes object.
type Validator interface {
	// Validate will received a pointer to an object, validators can be
	// grouped in chains, that's why a stop boolean to stop executing the chain
	// can be returned the validator, the valid parameter will denotate if the
	// object is valid (if not valid the chain will be stopped also) and a error.
	Validate(context.Context, metav1.Object) (stop bool, valid ValidatorResult, err error)
}

// ValidatorFunc is a helper type to create validators from functions.
type ValidatorFunc func(context.Context, metav1.Object) (stop bool, valid ValidatorResult, err error)

// Validate satisfies Validator interface.
func (f ValidatorFunc) Validate(ctx context.Context, obj metav1.Object) (stop bool, valid ValidatorResult, err error) {
	return f(ctx, obj)
}

// Chain is a chain of validators that will execute secuentially all the
// validators that have been added to it. It satisfies Mutator interface.
type Chain struct {
	validators []Validator
	logger     log.Logger
}

// NewChain returns a new chain.
func NewChain(logger log.Logger, validators ...Validator) *Chain {
	return &Chain{
		validators: validators,
		logger:     logger,
	}
}

// Validate will execute all the validation chain.
func (c *Chain) Validate(ctx context.Context, obj metav1.Object) (bool, ValidatorResult, error) {
	for _, vl := range c.validators {
		select {
		case <-ctx.Done():
			return false, ValidatorResult{}, fmt.Errorf("validator chain not finished correctly, context ended")
		default:
			stop, res, err := vl.Validate(ctx, obj)
			// If stop signal, or not valid or error return the obtained result and stop the chain.
			if stop || !res.Valid || err != nil {
				return true, res, err
			}
		}
	}

	// Return false if used a chain of chains.
	return false, ValidatorResult{Valid: true}, nil
}
