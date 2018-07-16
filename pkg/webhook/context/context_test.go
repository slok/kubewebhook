package context_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"

	whcontext "github.com/slok/kubewebhook/pkg/webhook/context"
)

func TestAdmissionRequestContext(t *testing.T) {
	tests := []struct {
		name  string
		ar    *admissionv1beta1.AdmissionRequest
		expAR *admissionv1beta1.AdmissionRequest
	}{
		{
			name:  "Missing admission review should return nil.",
			ar:    nil,
			expAR: nil,
		},

		{
			name: "Existing admission review should return the admission review.",
			ar: &admissionv1beta1.AdmissionRequest{
				Name:      "test",
				Namespace: "test2",
			},
			expAR: &admissionv1beta1.AdmissionRequest{
				Name:      "test",
				Namespace: "test2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := context.TODO()
			ctx = whcontext.SetAdmissionRequest(ctx, test.ar)
			gotAR := whcontext.GetAdmissionRequest(ctx)

			assert.Equal(test.expAR, gotAR)
		})
	}
}
