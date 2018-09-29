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

func TestIsAdmissionRequestDryRun(t *testing.T) {
	truep := true
	falsep := false

	tests := []struct {
		name      string
		ar        *admissionv1beta1.AdmissionRequest
		expResult bool
	}{
		{
			name:      "Missing admission review should return false.",
			ar:        nil,
			expResult: false,
		},
		{
			name: "Missing dry run in review should return false.",
			ar: &admissionv1beta1.AdmissionRequest{
				Name: "test",
			},
			expResult: false,
		},
		{
			name: "A dry run review should return true.",
			ar: &admissionv1beta1.AdmissionRequest{
				Name:   "test",
				DryRun: &truep,
			},
			expResult: true,
		},
		{
			name: "A not dry run review should return false.",
			ar: &admissionv1beta1.AdmissionRequest{
				Name:   "test",
				DryRun: &falsep,
			},
			expResult: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := context.TODO()
			ctx = whcontext.SetAdmissionRequest(ctx, test.ar)
			gotResult := whcontext.IsAdmissionRequestDryRun(ctx)

			assert.Equal(test.expResult, gotResult)
		})
	}
}
