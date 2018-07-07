package mutating

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/mattbaird/jsonpatch"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/webhook"
)

type dynamicWebhook struct {
	mutator      Mutator
	deserializer runtime.Decoder
	logger       log.Logger
}

// NewDynamicWebhook is the default implementation of a mutating webhook and will return a webhook ready
// for dynamic types that can receive different type of objects to mutate on the same webhook.
// This webhook will always allow the admission of the resource, only will deny in case of error.
func NewDynamicWebhook(mutator Mutator, logger log.Logger) webhook.Webhook {
	w := &dynamicWebhook{
		mutator: mutator,
		logger:  logger,
	}
	w.init()
	return w
}

func (w *dynamicWebhook) init() {
	// Register all the Kubernetes object types so we can receive any
	// kubernetes object and deserialize.
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	kubernetesscheme.AddToScheme(scheme)
	w.deserializer = codecs.UniversalDeserializer()
}

// MutatingAdmissionReview will handle the mutating of the admission review and
// return the AdmissionResponse.
func (w *dynamicWebhook) Review(ar *admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	w.logger.Debugf("reviewing request %s, named: %s/%s", ar.Request.UID, ar.Request.Namespace, ar.Request.Name)

	// Get the object.
	obj, _, err := w.deserializer.Decode(ar.Request.Object.Raw, nil, nil)
	if err != nil {
		return toAdmissionErrorResponse(fmt.Errorf("error deseralizing request raw object: %s", err), w.logger)
	}
	origObj, ok := obj.(metav1.Object)
	if !ok {
		err := fmt.Errorf("impossible to type assert the runtime.Object")
		return toAdmissionErrorResponse(err, w.logger)
	}

	// Copy the object to have the original and be able to get the patch.
	objCopy := obj.DeepCopyObject()
	mutatingObj, ok := objCopy.(metav1.Object)
	if !ok {
		err := fmt.Errorf("impossible to type assert the deep copy to metav1.Object")
		return toAdmissionErrorResponse(err, w.logger)
	}

	return mutatingAdmissionReview(w.mutator, ar.Request.UID, origObj, mutatingObj, w.logger)
}

type staticWebhook struct {
	objType      reflect.Type
	deserializer runtime.Decoder
	mutator      Mutator
	logger       log.Logger
}

// NewStaticWebhook is a mutating webhook and will return a webhook ready for a type of resource
// it will mutate the received resources.
// This webhook will always allow the admission of the resource, only will deny in case of error.
func NewStaticWebhook(mutator Mutator, obj metav1.Object, logger log.Logger) (webhook.Webhook, error) {
	// Object is an interface, we assume that is a pointer.
	// Get the indirect type of the object.
	objType := reflect.Indirect(reflect.ValueOf(obj)).Type()

	// Create a custom deserializer for the received admission review request.
	runtimeScheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(runtimeScheme)

	return &staticWebhook{
		objType:      objType,
		deserializer: codecs.UniversalDeserializer(),
		mutator:      mutator,
		logger:       logger,
	}, nil
}

// newObj returns a new object of webhook's object type.
func (w *staticWebhook) newObj() metav1.Object {
	// Create a new object of the webhook resource type
	// convert to ptr and typeassert to Kubernetes Object.
	var obj interface{}
	newObj := reflect.New(w.objType)
	obj = newObj.Interface()
	return obj.(metav1.Object)
}

func (w *staticWebhook) Review(ar *admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	w.logger.Debugf("reviewing request %s, named: %s/%s", ar.Request.UID, ar.Request.Namespace, ar.Request.Name)
	obj := w.newObj()
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		return toAdmissionErrorResponse(fmt.Errorf("could not type assert metav1.Object to runtime.Object"), w.logger)
	}

	// Get the object.
	_, _, err := w.deserializer.Decode(ar.Request.Object.Raw, nil, runtimeObj)
	if err != nil {
		return toAdmissionErrorResponse(fmt.Errorf("error deseralizing request raw object: %s", err), w.logger)
	}

	// Copy the object to have the original and be able to get the patch.
	objCopy := runtimeObj.DeepCopyObject()
	mutatingObj, ok := objCopy.(metav1.Object)
	if !ok {
		err := fmt.Errorf("impossible to type assert the deep copy to metav1.Object")
		return toAdmissionErrorResponse(err, w.logger)
	}

	return mutatingAdmissionReview(w.mutator, ar.Request.UID, obj, mutatingObj, w.logger)

}

func mutatingAdmissionReview(mutator Mutator, admissionRequestUID types.UID, obj, copyObj metav1.Object, logger log.Logger) *admissionv1beta1.AdmissionResponse {

	// Mutate the object.
	_, err := mutator.Mutate(context.TODO(), copyObj)
	if err != nil {
		return toAdmissionErrorResponse(err, logger)
	}

	// Get the diff patch of the original and mutated object.
	origJSON, err := json.Marshal(obj)
	if err != nil {
		return toAdmissionErrorResponse(err, logger)

	}
	mutatedJSON, err := json.Marshal(copyObj)
	if err != nil {
		return toAdmissionErrorResponse(err, logger)
	}

	patch, err := jsonpatch.CreatePatch(origJSON, mutatedJSON)
	if err != nil {
		return toAdmissionErrorResponse(err, logger)
	}

	marshalledPatch, err := json.Marshal(patch)
	if err != nil {
		return toAdmissionErrorResponse(err, logger)
	}
	logger.Debugf("json patch for request %s: %s", admissionRequestUID, string(marshalledPatch))

	// Forge response.
	return &admissionv1beta1.AdmissionResponse{
		UID:     admissionRequestUID,
		Allowed: true,
		Patch:   marshalledPatch,
		PatchType: func() *admissionv1beta1.PatchType {
			pt := admissionv1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func toAdmissionErrorResponse(err error, logger log.Logger) *admissionv1beta1.AdmissionResponse {
	logger.Errorf("admission webhook error: %s", err)
	return &admissionv1beta1.AdmissionResponse{Result: &metav1.Status{Message: err.Error()}}
}
