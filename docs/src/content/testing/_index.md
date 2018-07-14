+++
title = "Testing"
description = ""
weight = 3
alwaysopen = true
+++

# Testing

One of the most important things is to have a reliable software. Testing will help us increasing reliability.

Admission webhooks are hard to test mainly because of three things:

- It's mandatory to use TLS.
- Apiserver is the one that makes the calls.
- The servie needs to be in a cluster or available in a exposed host.

Kubewebhook is tested, this means that it's not necessary to test the infrastructure code, you just need to test your mutators and validators logic.

This doesn't mean that you could create integration tests, but integration tests should be less than unit test.

Lets make a testing example of a mutator.

## Example

The example is [here][test-example-url] also

Imagine a webhook like this. It only sets labels on every pod that it receives

```go
package mutatortesting

import (
	"context"

	corev1 "k8s.io/api/core/v1"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

  "github.com/slok/kubewebhook/pkg/webhook/mutating"
)

// PodLabeler is a mutator that will set labels on the received pods.
type PodLabeler struct {
	labels map[string]string
}

// NewPodLabeler returns a new PodLabeler initialized.
func NewPodLabeler(labels map[string]string) mutating.Mutator {
	if labels == nil {
		labels = make(map[string]string)
	}
	return &PodLabeler{
		labels: labels,
	}
}

// Mutate will set the required labels on the pods. Satisfies mutating.Mutator interface.
func (p *PodLabeler) Mutate(ctx context.Context, obj metav1.Object) (bool, error) {
	pod := obj.(*corev1.Pod)

	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	for k, v := range p.labels {
		pod.Labels[k] = v
	}
	return false, nil
}
```

You would make a test that check the business logic of the mutator itself and nothing more.

```go
package mutatortesting_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/kubewebhook/examples/mutatortesting"
)

func TestPodTaggerMutate(t *testing.T) {
	tests := []struct {
		name   string
		pod    *corev1.Pod
		labels map[string]string
		expPod *corev1.Pod
		expErr bool
	}{
		{
			name: "Mutating a pod without labels should set the labels correctly.",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			labels: map[string]string{"bruce": "wayne", "peter": "parker"},
			expPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test",
					Labels: map[string]string{"bruce": "wayne", "peter": "parker"},
				},
			},
		},
		{
			name: "Mutating a pod with labels should aggregate and replace the labels with the existing ones.",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test",
					Labels: map[string]string{"bruce": "banner", "tony": "stark"},
				},
			},
			labels: map[string]string{"bruce": "wayne", "peter": "parker"},
			expPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test",
					Labels: map[string]string{"bruce": "wayne", "peter": "parker", "tony": "stark"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			pl := mutatortesting.NewPodLabeler(test.labels)
			gotPod := test.pod
			_, err := pl.Mutate(context.TODO(), gotPod)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				// Check the expected pod.
				assert.Equal(test.expPod, gotPod)
			}
		})
	}
}
```

[test-example-url]: https://github.com/slok/kubewebhook/tree/master/examples/mutatortesting
