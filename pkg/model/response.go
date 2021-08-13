package model

// AdmissionResponse is the interface type that all the different
// types of webhooks must satisfy.
type AdmissionResponse interface {
	isWebhookResponse() bool // Sealed interface, we only want to abstract all webhook responses.
}

// ValidatingAdmissionResponse is the response for validating webhooks.
type ValidatingAdmissionResponse struct {
	admissionResponse

	ID       string
	Allowed  bool
	Message  string
	Warnings []string
}

// MutatingAdmissionResponse is the response for mutating webhooks.
type MutatingAdmissionResponse struct {
	admissionResponse

	ID             string
	JSONPatchPatch []byte
	Warnings       []string
}

// Helper type to satisfy the AdmissionResponse sealed interface.
type admissionResponse struct{}

func (admissionResponse) isWebhookResponse() bool { return true }
