package mutating_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/v2/pkg/log"
	"github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	"github.com/slok/kubewebhook/v2/pkg/webhook/mutating/mutatingmock"
)

func TestMutatorChain(t *testing.T) {
	tests := map[string]struct {
		name         string
		initalObj    metav1.Object
		mutatorMocks func() []mutating.Mutator
		expResult    *mutating.MutatorResult
		expErr       bool
	}{
		"Should call all the mutators.": {
			mutatorMocks: func() []mutating.Mutator {
				m1, m2, m3, m4, m5 := &mutatingmock.Mutator{}, &mutatingmock.Mutator{}, &mutatingmock.Mutator{}, &mutatingmock.Mutator{}, &mutatingmock.Mutator{}
				m1.On("Mutate", mock.Anything, mock.Anything, mock.Anything).Return(&mutating.MutatorResult{}, nil)
				m2.On("Mutate", mock.Anything, mock.Anything, mock.Anything).Return(&mutating.MutatorResult{}, nil)
				m3.On("Mutate", mock.Anything, mock.Anything, mock.Anything).Return(&mutating.MutatorResult{}, nil)
				m4.On("Mutate", mock.Anything, mock.Anything, mock.Anything).Return(&mutating.MutatorResult{}, nil)
				m5.On("Mutate", mock.Anything, mock.Anything, mock.Anything).Return(&mutating.MutatorResult{}, nil)
				return []mutating.Mutator{m1, m2, m3, m4, m5}
			},
			expResult: &mutating.MutatorResult{},
		},

		"Should call all the mutators and pass the previous object mutator, at last return the object of the latest mutator.": {
			initalObj: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p0"}},
			mutatorMocks: func() []mutating.Mutator {
				obj0 := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p0"}}
				obj1 := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p1"}}
				obj2 := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p2"}}
				obj3 := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p3"}}
				obj4 := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p4"}}

				m1, m2, m3, m4, m5 := &mutatingmock.Mutator{}, &mutatingmock.Mutator{}, &mutatingmock.Mutator{}, &mutatingmock.Mutator{}, &mutatingmock.Mutator{}
				m1.On("Mutate", mock.Anything, mock.Anything, obj0).Return(&mutating.MutatorResult{MutatedObject: obj1}, nil)
				m2.On("Mutate", mock.Anything, mock.Anything, obj1).Return(&mutating.MutatorResult{MutatedObject: obj2}, nil)
				m3.On("Mutate", mock.Anything, mock.Anything, obj2).Return(&mutating.MutatorResult{}, nil) // No mutation, should keep the previous one.
				m4.On("Mutate", mock.Anything, mock.Anything, obj2).Return(&mutating.MutatorResult{MutatedObject: obj3}, nil)
				m5.On("Mutate", mock.Anything, mock.Anything, obj3).Return(&mutating.MutatorResult{MutatedObject: obj4}, nil)
				return []mutating.Mutator{m1, m2, m3, m4, m5}
			},
			expResult: &mutating.MutatorResult{
				MutatedObject: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p4"}},
			},
		},

		"In case the last mutator doesn't return any object, the original one should be returned.": {
			initalObj: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p0"}},
			mutatorMocks: func() []mutating.Mutator {
				obj0 := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p0"}}
				m1 := &mutatingmock.Mutator{}
				m1.On("Mutate", mock.Anything, mock.Anything, obj0).Return(&mutating.MutatorResult{}, nil)
				return []mutating.Mutator{m1}
			},
			expResult: &mutating.MutatorResult{
				MutatedObject: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p0"}},
			},
		},

		"Should stop in the middle of the chain if any of the mutators stops the chain..": {
			mutatorMocks: func() []mutating.Mutator {
				m1, m2, m3, m4, m5 := &mutatingmock.Mutator{}, &mutatingmock.Mutator{}, &mutatingmock.Mutator{}, &mutatingmock.Mutator{}, &mutatingmock.Mutator{}
				m1.On("Mutate", mock.Anything, mock.Anything, mock.Anything).Return(&mutating.MutatorResult{}, nil)
				m2.On("Mutate", mock.Anything, mock.Anything, mock.Anything).Return(&mutating.MutatorResult{}, nil)
				m3.On("Mutate", mock.Anything, mock.Anything, mock.Anything).Return(&mutating.MutatorResult{StopChain: true}, nil)
				return []mutating.Mutator{m1, m2, m3, m4, m5}
			},
			expResult: &mutating.MutatorResult{StopChain: true},
		},

		"In case of error the chain should be stopped.": {
			mutatorMocks: func() []mutating.Mutator {
				m1, m2, m3, m4, m5 := &mutatingmock.Mutator{}, &mutatingmock.Mutator{}, &mutatingmock.Mutator{}, &mutatingmock.Mutator{}, &mutatingmock.Mutator{}
				m1.On("Mutate", mock.Anything, mock.Anything, mock.Anything).Return(&mutating.MutatorResult{}, nil)
				m2.On("Mutate", mock.Anything, mock.Anything, mock.Anything).Return(&mutating.MutatorResult{}, nil)
				m3.On("Mutate", mock.Anything, mock.Anything, mock.Anything).Return(&mutating.MutatorResult{}, fmt.Errorf("wanted error"))
				return []mutating.Mutator{m1, m2, m3, m4, m5}
			},
			expErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Mocks.
			mutators := test.mutatorMocks()

			// Execute.
			chain := mutating.NewChain(log.Dummy, mutators...)
			res, err := chain.Mutate(context.TODO(), nil, test.initalObj)

			// Check result.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expResult, res)
			}

			// Check calls where ok.
			for _, m := range mutators {
				mm := m.(*mutatingmock.Mutator)
				mm.AssertExpectations(t)
			}
		})
	}
}
