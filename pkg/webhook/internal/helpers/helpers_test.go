package helpers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/slok/kubewebhook/v2/pkg/webhook/internal/helpers"
)

type msi = map[string]interface{}

// Deployment.
var (
	rawYAMLDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-test
  namespace: "test-ns"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
`

	rawJSONDeployment = `
{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "name": "nginx-test",
    "namespace": "test-ns"
  },
  "spec": {
    "replicas": 3,
    "selector": {
      "matchLabels": {
        "app": "nginx"
      }
    },
    "template": {
      "metadata": {
        "labels": {
          "app": "nginx"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "nginx",
            "image": "nginx"
          }
        ]
      }
    }
  }
}
`

	k8sObjDeployment = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-test",
			Namespace: "test-ns",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &([]int32{3}[0]),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app": "nginx",
			}},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{Name: "nginx", Image: "nginx"},
					},
				},
			},
		},
	}

	unstructuredObjDeployment = &unstructured.Unstructured{
		Object: msi{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": msi{
				"name":      "nginx-test",
				"namespace": "test-ns",
			},

			"spec": msi{
				"replicas": int64(3),
				"selector": msi{
					"matchLabels": msi{
						"app": "nginx",
					},
				},
				"template": msi{
					"metadata": msi{
						"labels": msi{
							"app": "nginx",
						},
					},
					"spec": msi{
						"containers": []interface{}{
							msi{
								"name":  "nginx",
								"image": "nginx",
							},
						},
					},
				},
			},
		},
	}
)

// PodExecOptions (special K8s runtime.Object types that don't satisfy metav1.Object).
var (
	rawJSONPodExecOptions = `
{
   "kind":"PodExecOptions",
   "apiVersion":"v1",
   "stdin":true,
   "stdout":true,
   "tty":true,
   "container":"nginx",
   "command":[
      "/bin/sh"
   ]
}`
	UnstructuredObjPodExecOptions = &unstructured.Unstructured{
		Object: msi{
			"apiVersion": "v1",
			"kind":       "PodExecOptions",
			"stdin":      true,
			"stdout":     true,
			"tty":        true,
			"container":  "nginx",
			"command": []interface{}{
				"/bin/sh",
			},
		},
	}
)

func TestObjectCreator(t *testing.T) {
	tests := map[string]struct {
		objectCreator func() helpers.ObjectCreator
		raw           string
		expObj        helpers.K8sObject
		expErr        bool
	}{
		"Static with invalid objects should fail.": {
			objectCreator: func() helpers.ObjectCreator {
				return helpers.NewStaticObjectCreator(&appsv1.Deployment{})
			},
			raw:    "{",
			expErr: true,
		},

		"Static object creation with JSON raw data should return the object on the specific type.": {
			objectCreator: func() helpers.ObjectCreator {
				return helpers.NewStaticObjectCreator(&appsv1.Deployment{})
			},
			raw:    rawJSONDeployment,
			expObj: k8sObjDeployment,
		},

		"Static object creation with YAML raw data should return the object on the specific type.": {
			objectCreator: func() helpers.ObjectCreator {
				return helpers.NewStaticObjectCreator(&appsv1.Deployment{})
			},
			raw:    rawYAMLDeployment,
			expObj: k8sObjDeployment,
		},

		"Static with unstructured object creation should return the object on unstructured type.": {
			objectCreator: func() helpers.ObjectCreator {
				return helpers.NewStaticObjectCreator(&unstructured.Unstructured{})
			},
			raw:    rawYAMLDeployment,
			expObj: unstructuredObjDeployment,
		},

		"Dynamic with invalid objects should fail.": {
			objectCreator: func() helpers.ObjectCreator {
				return helpers.NewDynamicObjectCreator()
			},
			raw:    "{",
			expErr: true,
		},

		"Dynamic object creation with JSON should return the object on an inferred type.": {
			objectCreator: func() helpers.ObjectCreator {
				return helpers.NewDynamicObjectCreator()
			},
			raw:    rawJSONDeployment,
			expObj: k8sObjDeployment,
		},

		"Dynamic object creation with YAML should return the object on an inferred type.": {
			objectCreator: func() helpers.ObjectCreator {
				return helpers.NewDynamicObjectCreator()
			},
			raw:    rawYAMLDeployment,
			expObj: k8sObjDeployment,
		},

		"Static with unstructured using only runtime.Object compatible creation should return the object unstructured type.": {
			objectCreator: func() helpers.ObjectCreator {
				return helpers.NewStaticObjectCreator(&unstructured.Unstructured{})
			},
			raw:    rawJSONPodExecOptions,
			expObj: UnstructuredObjPodExecOptions,
		},

		"Dynamic only runtime.Object creation should return the object on an inferred unstructured type.": {
			objectCreator: func() helpers.ObjectCreator {
				return helpers.NewDynamicObjectCreator()
			},
			raw:    rawJSONPodExecOptions,
			expObj: UnstructuredObjPodExecOptions,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			oc := test.objectCreator()
			gotObj, err := oc.NewObject([]byte(test.raw))

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expObj, gotObj)
			}
		})
	}
}
