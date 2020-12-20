package validating_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/slok/kubewebhook/v2/pkg/log"
	"github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	"github.com/slok/kubewebhook/v2/pkg/webhook/validating/validatingmock"
)

func TestValidatorChain(t *testing.T) {
	tests := map[string]struct {
		validatorMocks func() []validating.Validator
		expResult      *validating.ValidatorResult
		expErr         bool
	}{
		"Should call all the validators if all the validators return that is valid.": {
			validatorMocks: func() []validating.Validator {
				m1, m2, m3, m4, m5 := &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}
				m1.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true}, nil)
				m2.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true}, nil)
				m3.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true}, nil)
				m4.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true}, nil)
				m5.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true}, nil)
				return []validating.Validator{m1, m2, m3, m4, m5}
			},
			expResult: &validating.ValidatorResult{Valid: true},
		},

		"Should stop in the middle of the chain and return the result of the stop.": {
			validatorMocks: func() []validating.Validator {
				m1, m2, m3, m4, m5 := &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}
				m1.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true}, nil)
				m2.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true}, nil)
				m3.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{StopChain: true, Valid: true}, nil)
				return []validating.Validator{m1, m2, m3, m4, m5}
			},
			expResult: &validating.ValidatorResult{StopChain: true, Valid: true},
		},

		"Should stop in the middle of the chain and return not valid when any validator returns not valid.": {
			validatorMocks: func() []validating.Validator {
				m1, m2, m3, m4, m5 := &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}
				m1.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true}, nil)
				m2.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true}, nil)
				m3.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: false, Message: "something"}, nil)
				return []validating.Validator{m1, m2, m3, m4, m5}
			},
			expResult: &validating.ValidatorResult{Valid: false, Message: "something"},
		},

		"In case of error, the chain should be stopped and return invalid.": {
			validatorMocks: func() []validating.Validator {
				m1, m2, m3, m4, m5 := &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}
				m1.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true}, nil)
				m2.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true}, nil)
				m3.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true}, fmt.Errorf("wanted error"))
				return []validating.Validator{m1, m2, m3, m4, m5}
			},
			expErr:    true,
			expResult: &validating.ValidatorResult{Valid: false},
		},

		"Warning messages shouldn't be lost in the chain.": {
			validatorMocks: func() []validating.Validator {
				m1, m2, m3, m4, m5 := &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}
				m1.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true, Warnings: []string{"w1"}}, nil)
				m2.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true, Warnings: []string{"w2"}}, nil)
				m3.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true, Warnings: []string{"w3", "w3.5"}}, nil)
				m4.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true, Warnings: []string{"w4"}}, nil)
				m5.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true, Warnings: []string{"w5"}}, nil)
				return []validating.Validator{m1, m2, m3, m4, m5}
			},
			expResult: &validating.ValidatorResult{
				Valid:    true,
				Warnings: []string{"w1", "w2", "w3", "w3.5", "w4", "w5"},
			},
		},

		"Warning messages shouldn't be lost in the chain (stopped chain by invalid).": {
			validatorMocks: func() []validating.Validator {
				m1, m2, m3, m4, m5 := &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}
				m1.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true, Warnings: []string{"w1"}}, nil)
				m2.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true, Warnings: []string{"w2"}}, nil)
				m3.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: false, Warnings: []string{"w3", "w3.5"}}, nil)
				return []validating.Validator{m1, m2, m3, m4, m5}
			},
			expResult: &validating.ValidatorResult{
				Valid:    false,
				Warnings: []string{"w1", "w2", "w3", "w3.5"},
			},
		},

		"Warning messages shouldn't be lost in the chain (stopped chain by stop chain flag).": {
			validatorMocks: func() []validating.Validator {
				m1, m2, m3, m4, m5 := &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}, &validatingmock.Validator{}
				m1.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true, Warnings: []string{"w1"}}, nil)
				m2.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{Valid: true, Warnings: []string{"w2"}}, nil)
				m3.On("Validate", mock.Anything, mock.Anything, mock.Anything).Return(&validating.ValidatorResult{StopChain: true, Valid: true, Warnings: []string{"w3", "w3.5"}}, nil)
				return []validating.Validator{m1, m2, m3, m4, m5}
			},
			expResult: &validating.ValidatorResult{
				StopChain: true,
				Valid:     true,
				Warnings:  []string{"w1", "w2", "w3", "w3.5"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			// Mocks.
			validators := test.validatorMocks()

			// Execute.
			chain := validating.NewChain(log.Dummy, validators...)
			res, err := chain.Validate(context.TODO(), nil, nil)

			// Check results.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expResult, res)
			}

			// Check validator calls.
			for _, m := range validators {
				mv := m.(*validatingmock.Validator)
				mv.AssertExpectations(t)
			}
		})
	}
}
