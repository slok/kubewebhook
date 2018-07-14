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
