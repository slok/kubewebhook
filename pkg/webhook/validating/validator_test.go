package validating_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mvalidating "github.com/slok/kubewebhook/mocks/webhook/validating"
	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/webhook/validating"
)

func TestValidatorChain(t *testing.T) {
	tests := []struct {
		name           string
		validatorMocks func() []validating.Validator
		expResult      validating.ValidatorResult
		expErr         bool
	}{
		{
			name: "Should call all the validators if all the validators return that is valid",
			validatorMocks: func() []validating.Validator {
				m1, m2, m3, m4, m5 := &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}
				m1.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				m2.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				m3.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				m4.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				m5.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				return []validating.Validator{m1, m2, m3, m4, m5}
			},
			expResult: validating.ValidatorResult{Valid: true},
		},
		{
			name: "Should stop in the middle of the chain and return the result of the stop (valid)",
			validatorMocks: func() []validating.Validator {
				m1, m2, m3, m4, m5 := &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}
				m1.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				m2.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				m3.On("Validate", mock.Anything, mock.Anything).Return(true, validating.ValidatorResult{Valid: true}, nil)
				return []validating.Validator{m1, m2, m3, m4, m5}
			},
			expResult: validating.ValidatorResult{Valid: true},
		},
		{
			name: "Should stop in the middle of the chain and return the result of the stop (not valid)",
			validatorMocks: func() []validating.Validator {
				m1, m2, m3, m4, m5 := &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}
				m1.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				m2.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				m3.On("Validate", mock.Anything, mock.Anything).Return(true, validating.ValidatorResult{Valid: false}, nil)
				return []validating.Validator{m1, m2, m3, m4, m5}
			},
			expResult: validating.ValidatorResult{Valid: false},
		},
		{
			name: "Should stop in the middle of the chain and return not valid when any validator returns not valid",
			validatorMocks: func() []validating.Validator {
				m1, m2, m3, m4, m5 := &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}
				m1.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				m2.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				m3.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: false}, nil)
				return []validating.Validator{m1, m2, m3, m4, m5}
			},
			expResult: validating.ValidatorResult{Valid: false},
		},
		{
			name: "Should return an error and stop the chain returning a valid",
			validatorMocks: func() []validating.Validator {
				m1, m2, m3, m4, m5 := &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}, &mvalidating.Validator{}
				m1.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				m2.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, nil)
				m3.On("Validate", mock.Anything, mock.Anything).Return(false, validating.ValidatorResult{Valid: true}, fmt.Errorf("wanted error"))
				return []validating.Validator{m1, m2, m3, m4, m5}
			},
			expErr:    true,
			expResult: validating.ValidatorResult{Valid: false},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Mocks.
			validators := test.validatorMocks()
			chain := validating.NewChain(log.Dummy, validators...)
			_, res, err := chain.Validate(context.TODO(), nil)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expResult, res)

				// Check calls where ok.
				for _, m := range validators {
					mv := m.(*mvalidating.Validator)
					mv.AssertExpectations(t)
				}
			}
		})
	}
}
