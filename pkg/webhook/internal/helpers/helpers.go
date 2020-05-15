package helpers

import (
	"fmt"
	"reflect"
	"strings"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/slok/kubewebhook/pkg/log"
)

// ToAdmissionErrorResponse transforms an error into a admission response with error.
func ToAdmissionErrorResponse(uid types.UID, err error, logger log.Logger) *admissionv1beta1.AdmissionResponse {
	logger.Errorf("admission webhook error: %s", err)
	return &admissionv1beta1.AdmissionResponse{
		UID: uid,
		Result: &metav1.Status{
			Message: err.Error(),
			Status:  metav1.StatusFailure,
		},
	}
}

// NewK8sObj returns a new object of a Kubernetes type based on the type.
func NewK8sObj(t reflect.Type) metav1.Object {
	// Create a new object of the webhook resource type
	// convert to ptr and typeassert to Kubernetes Object.
	var obj interface{}
	newObj := reflect.New(t)
	obj = newObj.Interface()
	return obj.(metav1.Object)
}

// GetK8sObjType returns the type (not the pointer type) of a kubernetes object.
func GetK8sObjType(obj metav1.Object) reflect.Type {
	// Object is an interface, is safe to assume that is a pointer.
	// Get the indirect type of the object.
	return reflect.Indirect(reflect.ValueOf(obj)).Type()
}

// GroupVersionResourceToString returns a string representation. It differs from the
// original stringer of the object itself.
func GroupVersionResourceToString(gvr metav1.GroupVersionResource) string {
	return strings.Join([]string{gvr.Group, "/", gvr.Version, "/", gvr.Resource}, "")
}

// ObjectCreator knows how to create objects from Raw JSON data into specific types.
type ObjectCreator interface {
	NewObject(rawJSON []byte) (runtime.Object, error)
}

type staticObjectCreator struct {
	objType      reflect.Type
	deserializer runtime.Decoder
}

// NewStaticObjectCreator doesn't need to infer the type, it will create a new schema and create a new
// object with the same type from the received object type.
func NewStaticObjectCreator(obj metav1.Object) ObjectCreator {
	codecs := serializer.NewCodecFactory(runtime.NewScheme())
	return staticObjectCreator{
		objType:      GetK8sObjType(obj),
		deserializer: codecs.UniversalDeserializer(),
	}
}

func (s staticObjectCreator) NewObject(rawJSON []byte) (runtime.Object, error) {
	runtimeObj, ok := NewK8sObj(s.objType).(runtime.Object)
	if !ok {
		return nil, fmt.Errorf("could not type assert metav1.Object to runtime.Object")
	}

	_, _, err := s.deserializer.Decode(rawJSON, nil, runtimeObj)
	if err != nil {
		return nil, fmt.Errorf("error deseralizing request raw object: %s", err)
	}

	return runtimeObj, nil
}

type dynamicObjectCreator struct {
	universalDecoder    runtime.Decoder
	unstructuredDecoder runtime.Decoder
}

// NewDynamicObjectCreator returns a object creator that knows how to return objects from raw
// JSON data without the need of knowing the type.
//
// To be able to infer the types the types need to be registered on the global client Scheme.
// Normally when a user tries casting the metav1.Object to a specific type, the object is already
// registered. In case the type is not registered and the object can't be created it will fallback
// to an Unstructured type.
//
// Useful to make dynamic webhooks that expect multiple or unknown types.
func NewDynamicObjectCreator() ObjectCreator {
	return dynamicObjectCreator{
		universalDecoder:    clientsetscheme.Codecs.UniversalDeserializer(),
		unstructuredDecoder: unstructured.UnstructuredJSONScheme,
	}
}

func (d dynamicObjectCreator) NewObject(rawJSON []byte) (runtime.Object, error) {
	runtimeObj, _, err := d.universalDecoder.Decode(rawJSON, nil, nil)
	// Fallback to unstructured.
	if err != nil {
		runtimeObj, _, err = d.unstructuredDecoder.Decode(rawJSON, nil, nil)
	}
	return runtimeObj, err
}
