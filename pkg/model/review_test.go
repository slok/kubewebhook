package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/slok/kubewebhook/v2/pkg/model"
)

var (
	falseBool = false
	trueBool  = true
)

func getBaseARV1Beta1() *admissionv1beta1.AdmissionReview {
	return &admissionv1beta1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1beta1",
		},
		Request: &admissionv1beta1.AdmissionRequest{
			Name:            "test-1",
			Namespace:       "ns-1",
			Kind:            metav1.GroupVersionKind{Group: "core", Kind: "Pod", Version: "v1"},
			RequestKind:     &metav1.GroupVersionKind{Group: "core", Kind: "Pod", Version: "v1"},
			RequestResource: &metav1.GroupVersionResource{Group: "core", Resource: "pods", Version: "v1"},
			UID:             "id-1",
			Operation:       admissionv1beta1.Create,
			UserInfo:        authenticationv1.UserInfo{},
			OldObject:       runtime.RawExtension{Raw: []byte("old-raw-thingy")},
			Object:          runtime.RawExtension{Raw: []byte("raw-thingy")},
			DryRun:          &trueBool,
		},
	}
}

func getBaseARV1Beta1WithoutOptional() *admissionv1beta1.AdmissionReview {
	return &admissionv1beta1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1beta1",
		},
		Request: &admissionv1beta1.AdmissionRequest{
			Name:      "test-1",
			Namespace: "ns-1",
			Kind:      metav1.GroupVersionKind{Group: "core", Kind: "Pod", Version: "v1"},
			Resource:  metav1.GroupVersionResource{Group: "core", Resource: "pods", Version: "v1"},
			UID:       "id-1",
			Operation: admissionv1beta1.Create,
			UserInfo:  authenticationv1.UserInfo{},
			OldObject: runtime.RawExtension{Raw: []byte("old-raw-thingy")},
			Object:    runtime.RawExtension{Raw: []byte("raw-thingy")},
			DryRun:    &trueBool,
		},
	}
}

func getBaseARV1() *admissionv1.AdmissionReview {
	return &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1beta1",
		},
		Request: &admissionv1.AdmissionRequest{
			Name:            "test-1",
			Namespace:       "ns-1",
			Kind:            metav1.GroupVersionKind{Group: "core", Kind: "Pod", Version: "v1"},
			RequestKind:     &metav1.GroupVersionKind{Group: "core", Kind: "Pod", Version: "v1"},
			RequestResource: &metav1.GroupVersionResource{Group: "core", Resource: "pods", Version: "v1"},
			UID:             "id-1",
			Operation:       admissionv1.Create,
			UserInfo:        authenticationv1.UserInfo{},
			OldObject:       runtime.RawExtension{Raw: []byte("old-raw-thingy")},
			Object:          runtime.RawExtension{Raw: []byte("raw-thingy")},
			DryRun:          &trueBool,
		},
	}
}

func getBaseModelV1Beta1() model.AdmissionReview {
	return model.AdmissionReview{
		OriginalAdmissionReview: getBaseARV1Beta1(),
		ID:                      "id-1",
		Name:                    "test-1",
		Namespace:               "ns-1",
		Version:                 model.AdmissionReviewVersionV1beta1,
		Operation:               model.OperationCreate,
		UserInfo:                authenticationv1.UserInfo{},
		OldObjectRaw:            []byte("old-raw-thingy"),
		NewObjectRaw:            []byte("raw-thingy"),
		RequestGVR:              &metav1.GroupVersionResource{Group: "core", Resource: "pods", Version: "v1"},
		RequestGVK:              &metav1.GroupVersionKind{Group: "core", Kind: "Pod", Version: "v1"},
		DryRun:                  true,
	}
}

func getBaseModelV1() model.AdmissionReview {
	return model.AdmissionReview{
		OriginalAdmissionReview: getBaseARV1(),
		ID:                      "id-1",
		Name:                    "test-1",
		Namespace:               "ns-1",
		Version:                 model.AdmissionReviewVersionV1,
		Operation:               model.OperationCreate,
		UserInfo:                authenticationv1.UserInfo{},
		OldObjectRaw:            []byte("old-raw-thingy"),
		NewObjectRaw:            []byte("raw-thingy"),
		RequestGVR:              &metav1.GroupVersionResource{Group: "core", Resource: "pods", Version: "v1"},
		RequestGVK:              &metav1.GroupVersionKind{Group: "core", Kind: "Pod", Version: "v1"},
		DryRun:                  true,
	}
}

func TestNewAdmissionReviewV1Beta1(t *testing.T) {
	tests := map[string]struct {
		ar       func() *admissionv1beta1.AdmissionReview
		expModel func() model.AdmissionReview
	}{
		"Regular Kubernetes object to model.": {
			ar:       getBaseARV1Beta1,
			expModel: getBaseModelV1Beta1,
		},

		"Regular Kubernetes object to model (false dry-run).": {
			ar: func() *admissionv1beta1.AdmissionReview {
				o := getBaseARV1Beta1()
				o.Request.DryRun = &falseBool
				return o
			},
			expModel: func() model.AdmissionReview {
				o := getBaseARV1Beta1()
				o.Request.DryRun = &falseBool

				m := getBaseModelV1Beta1()
				m.OriginalAdmissionReview = o
				m.DryRun = false
				return m
			},
		},

		"Regular Kubernetes object to model (Update op).": {
			ar: func() *admissionv1beta1.AdmissionReview {
				o := getBaseARV1Beta1()
				o.Request.Operation = admissionv1beta1.Update
				return o
			},
			expModel: func() model.AdmissionReview {
				o := getBaseARV1Beta1()
				o.Request.Operation = admissionv1beta1.Update

				m := getBaseModelV1Beta1()
				m.OriginalAdmissionReview = o
				m.Operation = model.OperationUpdate
				return m
			},
		},

		"Regular Kubernetes object to model (Delete op).": {
			ar: func() *admissionv1beta1.AdmissionReview {
				o := getBaseARV1Beta1()
				o.Request.Operation = admissionv1beta1.Delete
				return o
			},
			expModel: func() model.AdmissionReview {
				o := getBaseARV1Beta1()
				o.Request.Operation = admissionv1beta1.Delete

				m := getBaseModelV1Beta1()
				m.OriginalAdmissionReview = o
				m.Operation = model.OperationDelete
				return m
			},
		},

		"Regular Kubernetes object to model (Connect op).": {
			ar: func() *admissionv1beta1.AdmissionReview {
				o := getBaseARV1Beta1()
				o.Request.Operation = admissionv1beta1.Connect
				return o
			},
			expModel: func() model.AdmissionReview {
				o := getBaseARV1Beta1()
				o.Request.Operation = admissionv1beta1.Connect

				m := getBaseModelV1Beta1()
				m.OriginalAdmissionReview = o
				m.Operation = model.OperationConnect
				return m
			},
		},

		"Regular Kubernetes object to model without optional (RequestKind/RequestResource) (Connect op).": {
			ar: func() *admissionv1beta1.AdmissionReview {
				o := getBaseARV1Beta1WithoutOptional()
				o.Request.Operation = admissionv1beta1.Connect
				return o
			},
			expModel: func() model.AdmissionReview {
				o := getBaseARV1Beta1WithoutOptional()
				o.Request.Operation = admissionv1beta1.Connect

				m := getBaseModelV1Beta1()
				m.OriginalAdmissionReview = o
				m.Operation = model.OperationConnect
				return m
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotModel := model.NewAdmissionReviewV1Beta1(test.ar())
			assert.Equal(t, test.expModel(), gotModel)
		})
	}
}

func TestNewAdmissionReviewV1(t *testing.T) {
	tests := map[string]struct {
		ar       func() *admissionv1.AdmissionReview
		expModel func() model.AdmissionReview
	}{
		"Regular Kubernetes object to model.": {
			ar:       getBaseARV1,
			expModel: getBaseModelV1,
		},

		"Regular Kubernetes object to model (false dry-run).": {
			ar: func() *admissionv1.AdmissionReview {
				o := getBaseARV1()
				o.Request.DryRun = &falseBool
				return o
			},
			expModel: func() model.AdmissionReview {
				o := getBaseARV1()
				o.Request.DryRun = &falseBool

				m := getBaseModelV1()
				m.OriginalAdmissionReview = o
				m.DryRun = false
				return m
			},
		},

		"Regular Kubernetes object to model (Update op).": {
			ar: func() *admissionv1.AdmissionReview {
				o := getBaseARV1()
				o.Request.Operation = admissionv1.Update
				return o
			},
			expModel: func() model.AdmissionReview {
				o := getBaseARV1()
				o.Request.Operation = admissionv1.Update

				m := getBaseModelV1()
				m.OriginalAdmissionReview = o
				m.Operation = model.OperationUpdate
				return m
			},
		},

		"Regular Kubernetes object to model (Delete op).": {
			ar: func() *admissionv1.AdmissionReview {
				o := getBaseARV1()
				o.Request.Operation = admissionv1.Delete
				return o
			},
			expModel: func() model.AdmissionReview {
				o := getBaseARV1()
				o.Request.Operation = admissionv1.Delete

				m := getBaseModelV1()
				m.OriginalAdmissionReview = o
				m.Operation = model.OperationDelete
				return m
			},
		},

		"Regular Kubernetes object to model (Connect op).": {
			ar: func() *admissionv1.AdmissionReview {
				o := getBaseARV1()
				o.Request.Operation = admissionv1.Connect
				return o
			},
			expModel: func() model.AdmissionReview {
				o := getBaseARV1()
				o.Request.Operation = admissionv1.Connect

				m := getBaseModelV1()
				m.OriginalAdmissionReview = o
				m.Operation = model.OperationConnect
				return m
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotModel := model.NewAdmissionReviewV1(test.ar())
			assert.Equal(t, test.expModel(), gotModel)
		})
	}
}
