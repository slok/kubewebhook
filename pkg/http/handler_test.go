package http_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	mwebhook "github.com/slok/kubewebhook/mocks/webhook"
	kubewebhookhttp "github.com/slok/kubewebhook/pkg/http"
)

func getTestAdmissionReviewRequestStr(uid string) string {
	ar := admissionv1beta1.AdmissionReview{
		Request: &admissionv1beta1.AdmissionRequest{
			UID: types.UID(uid),
		},
	}
	jsonAR, _ := json.Marshal(ar)
	return string(jsonAR)
}

func TestDefaultWebhookFlow(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		reviewResponse *admissionv1beta1.AdmissionResponse
		expCode        int
		expBody        string
	}{
		{
			name:    "No admission review on request should return error",
			body:    "",
			expBody: "no body found\n",
			expCode: 400,
		},
		{
			name:    "Bad admission review on request should return error",
			body:    "wrong body",
			expBody: "could not decode the admission review from the request\n",
			expCode: 400,
		},
		{
			name: "A regular call to the webhook handler should execute the webhook and return OK if nothing failed",
			body: getTestAdmissionReviewRequestStr("1234567890"),
			reviewResponse: &admissionv1beta1.AdmissionResponse{
				UID:     "1234567890",
				Allowed: true,
			},
			expBody: `{"response":{"uid":"1234567890","allowed":true}}`,
			expCode: 200,
		},
		{
			name: "A regular call to the webhook handler should execute the webhook and return error if something failed",
			body: getTestAdmissionReviewRequestStr("1234567890"),
			reviewResponse: &admissionv1beta1.AdmissionResponse{
				UID: "1234567890",
				Result: &metav1.Status{
					Status:  "Failure",
					Message: "wanted error",
				},
			},
			expBody: `{"response":{"uid":"1234567890","allowed":false,"status":{"metadata":{},"status":"Failure","message":"wanted error"}}}`,
			expCode: 500,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Mocks.
			mwh := &mwebhook.Webhook{}
			mwh.On("Review", mock.Anything, mock.Anything).Once().Return(test.reviewResponse, nil)

			h, err := kubewebhookhttp.HandlerFor(mwh)
			require.NoError(err)

			req := httptest.NewRequest("GET", "/awesome/webhook", bytes.NewBufferString(test.body))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			assert.Equal(test.expCode, w.Code)
			assert.Equal(test.expBody, w.Body.String())
		})
	}
}
