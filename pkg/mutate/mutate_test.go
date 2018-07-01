package mutate_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mmutate "github.com/slok/kutator/mocks/mutate"
	"github.com/slok/kutator/pkg/log"
	"github.com/slok/kutator/pkg/mutate"
)

func TestMutatorChain(t *testing.T) {
	tests := []struct {
		name         string
		mutatorMocks func() []mutate.Mutator
		expErr       bool
	}{
		{
			name: "Should call all the mutators",
			mutatorMocks: func() []mutate.Mutator {
				m1, m2, m3, m4, m5 := &mmutate.Mutator{}, &mmutate.Mutator{}, &mmutate.Mutator{}, &mmutate.Mutator{}, &mmutate.Mutator{}
				m1.On("Mutate", mock.Anything, mock.Anything).Return(false, nil)
				m2.On("Mutate", mock.Anything, mock.Anything).Return(false, nil)
				m3.On("Mutate", mock.Anything, mock.Anything).Return(false, nil)
				m4.On("Mutate", mock.Anything, mock.Anything).Return(false, nil)
				m5.On("Mutate", mock.Anything, mock.Anything).Return(false, nil)
				return []mutate.Mutator{m1, m2, m3, m4, m5}
			},
		},
		{
			name: "Should stop in the middle of the chain",
			mutatorMocks: func() []mutate.Mutator {
				m1, m2, m3, m4, m5 := &mmutate.Mutator{}, &mmutate.Mutator{}, &mmutate.Mutator{}, &mmutate.Mutator{}, &mmutate.Mutator{}
				m1.On("Mutate", mock.Anything, mock.Anything).Return(false, nil)
				m2.On("Mutate", mock.Anything, mock.Anything).Return(false, nil)
				m3.On("Mutate", mock.Anything, mock.Anything).Return(true, nil)
				return []mutate.Mutator{m1, m2, m3, m4, m5}
			},
		},
		{
			name: "Should return an error and stop the chain",
			mutatorMocks: func() []mutate.Mutator {
				m1, m2, m3, m4, m5 := &mmutate.Mutator{}, &mmutate.Mutator{}, &mmutate.Mutator{}, &mmutate.Mutator{}, &mmutate.Mutator{}
				m1.On("Mutate", mock.Anything, mock.Anything).Return(false, nil)
				m2.On("Mutate", mock.Anything, mock.Anything).Return(false, nil)
				m3.On("Mutate", mock.Anything, mock.Anything).Return(false, fmt.Errorf("wanted error"))
				return []mutate.Mutator{m1, m2, m3, m4, m5}
			},
			expErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Mocks.
			mutators := test.mutatorMocks()
			chain := mutate.NewChain(log.Dummy, mutators...)
			_, err := chain.Mutate(context.TODO(), nil)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				// Check calls where ok.
				for _, m := range mutators {
					mm := m.(*mmutate.Mutator)
					mm.AssertExpectations(t)
				}
			}
		})
	}
}
