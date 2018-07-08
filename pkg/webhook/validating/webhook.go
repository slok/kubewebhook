package validating

import (
	"context"
	"fmt"
	"reflect"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/internal/helpers"
)

type staticwebhook struct {
	objType      reflect.Type
	deserializer runtime.Decoder
	validator    Validator
	logger       log.Logger
}

// NewWebhook is a validating webhook and will return a webhook ready for a type of resource
// it will validate the received resources.
func NewWebhook(validator Validator, obj metav1.Object, logger log.Logger) (webhook.Webhook, error) {
	// Create a custom deserializer for the received admission review request.
	runtimeScheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(runtimeScheme)

	return &staticwebhook{
		objType:      helpers.GetK8sObjType(obj),
		deserializer: codecs.UniversalDeserializer(),
		validator:    validator,
		logger:       logger,
	}, nil
}

func (w *staticwebhook) Review(ar *admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	uid := ar.Request.UID

	w.logger.Debugf("reviewing request %s, named: %s/%s", ar.Request.UID, ar.Request.Namespace, ar.Request.Name)

	obj := helpers.NewK8sObj(w.objType)
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		return helpers.ToAdmissionErrorResponse(uid, fmt.Errorf("could not type assert metav1.Object to runtime.Object"), w.logger)
	}

	// Get the object.
	_, _, err := w.deserializer.Decode(ar.Request.Object.Raw, nil, runtimeObj)
	if err != nil {
		return helpers.ToAdmissionErrorResponse(uid, fmt.Errorf("error deseralizing request raw object: %s", err), w.logger)
	}

	// Check validation on the object.
	_, res, err := w.validator.Validate(context.TODO(), obj)
	if err != nil {
		return helpers.ToAdmissionErrorResponse(uid, err, w.logger)
	}

	// Forge response.
	return &admissionv1beta1.AdmissionResponse{
		UID:     ar.Request.UID,
		Allowed: res.Valid,
		Result: &metav1.Status{
			Status:  metav1.StatusSuccess,
			Message: res.Message,
		},
	}
}
