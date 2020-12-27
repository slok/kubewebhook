package http_test

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	kubewebhookhttp "github.com/slok/kubewebhook/v2/pkg/http"
	"github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/webhook/webhookmock"
)

// `\n` in utf8, used to avoid scaping problems on body responses.
const newLine = "\u000A"

var encoder = json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, scheme.Scheme, json.SerializerOptions{Pretty: true})

func getTestAdmissionReviewV1beta1RequestStr(uid string) string {
	ar := &admissionv1beta1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1beta1",
		},
		Request: &admissionv1beta1.AdmissionRequest{
			Kind: metav1.GroupVersionKind{
				Group:   "core",
				Kind:    "Pod",
				Version: "v1",
			},
			UID: types.UID(uid),
			Object: runtime.RawExtension{
				Object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
			},
		},
	}
	var b bytes.Buffer
	_ = encoder.Encode(ar, &b)

	return b.String()
}

func getTestAdmissionReviewV1RequestStr(uid string) string {
	ar := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Request: &admissionv1.AdmissionRequest{
			Kind: metav1.GroupVersionKind{
				Group:   "core",
				Kind:    "Pod",
				Version: "v1",
			},
			UID: types.UID(uid),
			Object: runtime.RawExtension{
				Object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
			},
		},
	}
	var b bytes.Buffer
	_ = encoder.Encode(ar, &b)

	return b.String()
}

func TestDefaultWebhookFlow(t *testing.T) {
	tests := map[string]struct {
		name           string
		body           string
		mock           func(mw *webhookmock.Webhook)
		reviewResponse *model.AdmissionResponse
		expCode        int
		expBody        string
	}{
		"No admission review on request should return error": {
			body:    "",
			mock:    func(mw *webhookmock.Webhook) {},
			expBody: "no body found\n",
			expCode: 400,
		},

		"Bad admission review on request should return error": {
			body:    "wrong body",
			mock:    func(mw *webhookmock.Webhook) {},
			expBody: `could not decode the admission review from the request: couldn't get version/kind; json parse error: json: cannot unmarshal string into Go value of type struct { APIVersion string "json:\"apiVersion,omitempty\""; Kind string "json:\"kind,omitempty\"" }` + newLine,
			expCode: 400,
		},

		"A correct validation admission v1beta1 webhook that allows should not fail.": {
			body: getTestAdmissionReviewV1beta1RequestStr("1234567890"),
			mock: func(mw *webhookmock.Webhook) {
				resp := &model.ValidatingAdmissionResponse{
					ID:       "1234567890",
					Allowed:  true,
					Warnings: []string{"warn1", "warn2"}, // v1beta1 ignores warnings.
				}
				mw.On("Review", mock.Anything, mock.Anything).Once().Return(resp, nil)
			},
			expBody: `{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1beta1","response":{"uid":"1234567890","allowed":true}}`,
			expCode: 200,
		},

		"A correct validation admission v1beta1 webhook that doesn't allow should not fail.": {
			body: getTestAdmissionReviewV1beta1RequestStr("1234567890"),
			mock: func(mw *webhookmock.Webhook) {
				resp := &model.ValidatingAdmissionResponse{
					ID:       "1234567890",
					Allowed:  false,
					Message:  "this is not valid because reasons",
					Warnings: []string{"warn1", "warn2"}, // v1beta1 ignores warnings.
				}
				mw.On("Review", mock.Anything, mock.Anything).Once().Return(resp, nil)
			},
			expBody: `{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1beta1","response":{"uid":"1234567890","allowed":false,"status":{"metadata":{},"status":"Failure","message":"this is not valid because reasons","code":400}}}`,
			expCode: 200,
		},

		"A correct mutating admission v1beta1 webhook with mutation should not fail.": {
			body: getTestAdmissionReviewV1beta1RequestStr("1234567890"),
			mock: func(mw *webhookmock.Webhook) {
				resp := &model.MutatingAdmissionResponse{
					ID:             "1234567890",
					JSONPatchPatch: []byte(`{"something": something}`),
					Warnings:       []string{"warn1", "warn2"}, // v1beta1 ignores warnings.
				}
				mw.On("Review", mock.Anything, mock.Anything).Once().Return(resp, nil)
			},
			expBody: `{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1beta1","response":{"uid":"1234567890","allowed":true,"patch":"eyJzb21ldGhpbmciOiBzb21ldGhpbmd9","patchType":"JSONPatch"}}`,
			expCode: 200,
		},

		"A correct mutating admission v1beta1 webhook without mutation should not fail.": {
			body: getTestAdmissionReviewV1beta1RequestStr("1234567890"),
			mock: func(mw *webhookmock.Webhook) {
				resp := &model.MutatingAdmissionResponse{
					ID:             "1234567890",
					JSONPatchPatch: []byte(``),
					Warnings:       []string{"warn1", "warn2"}, // v1beta1 ignores warnings.
				}
				mw.On("Review", mock.Anything, mock.Anything).Once().Return(resp, nil)
			},
			expBody: `{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1beta1","response":{"uid":"1234567890","allowed":true,"patchType":"JSONPatch"}}`,
			expCode: 200,
		},

		"A correct validation admission v1 webhook that allows should not fail.": {
			body: getTestAdmissionReviewV1RequestStr("1234567890"),
			mock: func(mw *webhookmock.Webhook) {
				resp := &model.ValidatingAdmissionResponse{
					ID:       "1234567890",
					Allowed:  true,
					Warnings: []string{"warn1", "warn2"}, // v1beta1 ignores warnings.
				}
				mw.On("Review", mock.Anything, mock.Anything).Once().Return(resp, nil)
			},
			expBody: `{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1","response":{"uid":"1234567890","allowed":true,"warnings":["warn1","warn2"]}}`,
			expCode: 200,
		},

		"A correct validation admission v1 webhook that doesn't allow should not fail.": {
			body: getTestAdmissionReviewV1RequestStr("1234567890"),
			mock: func(mw *webhookmock.Webhook) {
				resp := &model.ValidatingAdmissionResponse{
					ID:       "1234567890",
					Allowed:  false,
					Message:  "this is not valid because reasons",
					Warnings: []string{"warn1", "warn2"}, // v1beta1 ignores warnings.
				}
				mw.On("Review", mock.Anything, mock.Anything).Once().Return(resp, nil)
			},
			expBody: `{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1","response":{"uid":"1234567890","allowed":false,"status":{"metadata":{},"status":"Failure","message":"this is not valid because reasons","code":400},"warnings":["warn1","warn2"]}}`,
			expCode: 200,
		},

		"A correct mutating admission v1 webhook should not fail.": {
			body: getTestAdmissionReviewV1RequestStr("1234567890"),
			mock: func(mw *webhookmock.Webhook) {
				resp := &model.MutatingAdmissionResponse{
					ID:             "1234567890",
					JSONPatchPatch: []byte(`{"something": something}`),
					Warnings:       []string{"warn1", "warn2"}, // v1beta1 ignores warnings.
				}
				mw.On("Review", mock.Anything, mock.Anything).Once().Return(resp, nil)
			},
			expBody: `{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1","response":{"uid":"1234567890","allowed":true,"patch":"eyJzb21ldGhpbmciOiBzb21ldGhpbmd9","patchType":"JSONPatch","warnings":["warn1","warn2"]}}`,
			expCode: 200,
		},

		"A correct mutating admission v1 webhook without mutation should not fail.": {
			body: getTestAdmissionReviewV1RequestStr("1234567890"),
			mock: func(mw *webhookmock.Webhook) {
				resp := &model.MutatingAdmissionResponse{
					ID:             "1234567890",
					JSONPatchPatch: []byte(``),
					Warnings:       []string{"warn1", "warn2"}, // v1beta1 ignores warnings.
				}
				mw.On("Review", mock.Anything, mock.Anything).Once().Return(resp, nil)
			},
			expBody: `{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1","response":{"uid":"1234567890","allowed":true,"patchType":"JSONPatch","warnings":["warn1","warn2"]}}`,
			expCode: 200,
		},

		"A regular mutating admission v1beta1 call to the webhook handler should execute the webhook and return error if something failed": {
			body: getTestAdmissionReviewV1beta1RequestStr("1234567890"),
			mock: func(mw *webhookmock.Webhook) {
				mw.On("Review", mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expBody: `{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1beta1","response":{"uid":"1234567890","allowed":false,"status":{"metadata":{},"status":"Failure","message":"wanted error"}}}`,
			expCode: 500,
		},

		"A regular mutating admission v1 call to the webhook handler should execute the webhook and return error if something failed": {
			body: getTestAdmissionReviewV1RequestStr("1234567890"),
			mock: func(mw *webhookmock.Webhook) {
				mw.On("Review", mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expBody: `{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1","response":{"uid":"1234567890","allowed":false,"status":{"metadata":{},"status":"Failure","message":"wanted error"}}}`,
			expCode: 500,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Mocks.
			mwh := &webhookmock.Webhook{}
			test.mock(mwh)
			mwh.On("ID").Maybe().Return("")
			mwh.On("Kind").Maybe().Return(model.WebhookKind(""))

			h, err := kubewebhookhttp.HandlerFor(kubewebhookhttp.HandlerConfig{Webhook: mwh})
			require.NoError(err)

			req := httptest.NewRequest("GET", "/awesome/webhook", bytes.NewBufferString(test.body))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			assert.Equal(test.expCode, w.Code)
			assert.Equal(test.expBody, w.Body.String())
		})
	}
}
