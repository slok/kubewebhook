package mutating

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/appscode/jsonpatch"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/internal/helpers"
)

type staticWebhook struct {
	objType      reflect.Type
	deserializer runtime.Decoder
	mutator      Mutator
	logger       log.Logger
}

// NewWebhook is a mutating webhook and will return a webhook ready for a type of resource.
// It will mutate the received resources.
// This webhook will always allow the admission of the resource, only will deny in case of error.
func NewWebhook(mutator Mutator, obj metav1.Object, logger log.Logger) (webhook.Webhook, error) {
	// Create a custom deserializer for the received admission review request.
	runtimeScheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(runtimeScheme)

	return &staticWebhook{
		objType:      helpers.GetK8sObjType(obj),
		deserializer: codecs.UniversalDeserializer(),
		mutator:      mutator,
		logger:       logger,
	}, nil
}

func (w *staticWebhook) Review(ctx context.Context, ar *admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	uid := ar.Request.UID

	w.logger.Debugf("reviewing request %s, named: %s/%s", uid, ar.Request.Namespace, ar.Request.Name)
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

	// Copy the object to have the original and be able to get the patch.
	objCopy := runtimeObj.DeepCopyObject()
	mutatingObj, ok := objCopy.(metav1.Object)
	if !ok {
		err := fmt.Errorf("impossible to type assert the deep copy to metav1.Object")
		return helpers.ToAdmissionErrorResponse(uid, err, w.logger)
	}

	return w.mutatingAdmissionReview(ctx, uid, obj, mutatingObj)

}

func (w *staticWebhook) mutatingAdmissionReview(ctx context.Context, admissionRequestUID types.UID, obj, copyObj metav1.Object) *admissionv1beta1.AdmissionResponse {

	// Mutate the object.
	_, err := w.mutator.Mutate(ctx, copyObj)
	if err != nil {
		return helpers.ToAdmissionErrorResponse(admissionRequestUID, err, w.logger)
	}

	// Get the diff patch of the original and mutated object.
	origJSON, err := json.Marshal(obj)
	if err != nil {
		return helpers.ToAdmissionErrorResponse(admissionRequestUID, err, w.logger)

	}
	mutatedJSON, err := json.Marshal(copyObj)
	if err != nil {
		return helpers.ToAdmissionErrorResponse(admissionRequestUID, err, w.logger)
	}

	patch, err := jsonpatch.CreatePatch(origJSON, mutatedJSON)
	if err != nil {
		return helpers.ToAdmissionErrorResponse(admissionRequestUID, err, w.logger)
	}

	marshalledPatch, err := json.Marshal(patch)
	if err != nil {
		return helpers.ToAdmissionErrorResponse(admissionRequestUID, err, w.logger)
	}
	w.logger.Debugf("json patch for request %s: %s", admissionRequestUID, string(marshalledPatch))

	// Forge response.
	return &admissionv1beta1.AdmissionResponse{
		UID:       admissionRequestUID,
		Allowed:   true,
		Patch:     marshalledPatch,
		PatchType: jsonPatchType,
	}
}

// jsonPatchType is the type for Kubernetes responses type.
var jsonPatchType = func() *admissionv1beta1.PatchType {
	pt := admissionv1beta1.PatchTypeJSONPatch
	return &pt
}()
