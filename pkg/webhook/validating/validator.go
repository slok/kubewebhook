package validating

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/pkg/log"
)

// ValidatorResult is the result of a validator.
type ValidatorResult struct {
	// StopChain will stop the chain of validators in case there is a chain set.
	StopChain bool
	// Valid tells the apiserver that the resource is correct and it should allow or not.
	Valid bool
	// Message will be used by the apiserver to give more information in case the resource is not valid.
	Message string
	// Warnings are special messages that can be set to warn the user (e.g deprecation messages, almost invalid resources...).
	Warnings []string
}

// Validator knows how to validate the received kubernetes object.
type Validator interface {
	// Validate will received a pointer to an object, validators can be
	// grouped in chains, that's why we have a `StopChain` boolean in the result,
	// to stop executing the validators chain.
	Validate(context.Context, metav1.Object) (result *ValidatorResult, err error)
}

//go:generate mockery --case underscore --output validatingmock --outpkg validatingmock --name Validator

// ValidatorFunc is a helper type to create validators from functions.
type ValidatorFunc func(context.Context, metav1.Object) (result *ValidatorResult, err error)

// Validate satisfies Validator interface.
func (f ValidatorFunc) Validate(ctx context.Context, obj metav1.Object) (result *ValidatorResult, err error) {
	return f(ctx, obj)
}

type chain struct {
	validators []Validator
	logger     log.Logger
}

// NewChain returns a new chain of validators.
// - If any of the validators returns an error, the chain will end.
// - If any of the validators returns an stopChain == true, the chain will end.
// - If any of the validators returns as no valid, the chain will end.
func NewChain(logger log.Logger, validators ...Validator) Validator {
	return chain{
		validators: validators,
		logger:     logger,
	}
}

// Validate will execute all the validation chain.
func (c chain) Validate(ctx context.Context, obj metav1.Object) (*ValidatorResult, error) {
	var warnings []string
	for _, vl := range c.validators {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("validator chain not finished correctly, context done")
		default:
			res, err := vl.Validate(ctx, obj)
			if err != nil {
				return nil, err
			}

			if res == nil {
				return nil, fmt.Errorf("validator result can't be `nil`")
			}

			// Don't lose the warnings through the chain.
			warnings = append(warnings, res.Warnings...)

			if res.StopChain || !res.Valid {
				res.Warnings = warnings
				return res, nil
			}
		}
	}

	return &ValidatorResult{
		Valid:    true,
		Warnings: warnings,
	}, nil
}
