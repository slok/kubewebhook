package context

import (
	"context"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
)

type contextKey struct {
	name string
}

var AdmissionRequestKey = &contextKey{"admissionRequest"}

// SetAdmissionRequest will set a admission request on the context and return the new context that has
// the admission request set.
func SetAdmissionRequest(ctx context.Context, ar *admissionv1beta1.AdmissionRequest) context.Context {
	return context.WithValue(ctx, AdmissionRequestKey, ar)
}

// GetAdmissionRequest returns the admission request stored on the context. If there is no admission
// request on the context it will return nil.
func GetAdmissionRequest(ctx context.Context) *admissionv1beta1.AdmissionRequest {
	val := ctx.Value(AdmissionRequestKey)
	if ar, ok := val.(*admissionv1beta1.AdmissionRequest); ok {
		return ar
	}
	return nil
}

// IsAdmissionRequestDryRun returns true if the admission request stored
// on the context is in dry run mode. If the request is missing or the
// request is not dry run then it will return false.
func IsAdmissionRequestDryRun(ctx context.Context) bool {
	ar := GetAdmissionRequest(ctx)
	if ar == nil {
		return false
	}

	if ar.DryRun == nil {
		return false
	}

	return *ar.DryRun
}
