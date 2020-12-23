// +build integration

package webhook_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	arv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"

	whhttp "github.com/slok/kubewebhook/v2/pkg/http"
	"github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/webhook"
	"github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	buildingv1 "github.com/slok/kubewebhook/v2/test/integration/crd/apis/building/v1"
	kubewebhookcrd "github.com/slok/kubewebhook/v2/test/integration/crd/client/clientset/versioned"
	helpercli "github.com/slok/kubewebhook/v2/test/integration/helper/cli"
	helperconfig "github.com/slok/kubewebhook/v2/test/integration/helper/config"
)

// testValidatingWebhookCommon tests the common use cases that should be shared among all webhook versions
// so the version of the webhook (v1 or v1beta1) should not make a difference.
func testValidatingWebhookCommon(t *testing.T, version string) {
	cfg := helperconfig.GetTestEnvConfig(t)

	cli, err := helpercli.GetK8sSTDClients(cfg.KubeConfigPath, nil)
	require.NoError(t, err, "error getting kubernetes client")
	crdcli, err := helpercli.GetK8sCRDClients(cfg.KubeConfigPath, nil)
	require.NoError(t, err, "error getting kubernetes CRD client")

	tests := map[string]struct {
		webhookRegisterCfg *arv1.ValidatingWebhookConfiguration
		webhook            func() webhook.Webhook
		execTest           func(t *testing.T, cli kubernetes.Interface, crdcli kubewebhookcrd.Interface)
	}{
		"(invalid, static, core) Having a static webhook, a validating webhook should not allow creating the pod and return a message.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesPod}, []string{version}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
					return &validating.ValidatorResult{
						Valid:   false,
						Message: "test message from validator",
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					ID:        "pod-validating-label",
					Obj:       &corev1.Pod{},
					Validator: val,
				})
				return vwh
			},
			execTest: func(t *testing.T, cli kubernetes.Interface, _ kubewebhookcrd.Interface) {
				// Crate a pod and check expectations.
				p := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "test", Image: "wrong"}},
					},
				}
				_, err := cli.CoreV1().Pods(p.Namespace).Create(context.TODO(), p, metav1.CreateOptions{})
				if assert.Error(t, err) {
					sErr, ok := err.(*apierrors.StatusError)
					if assert.True(t, ok) {
						assert.Equal(t, `admission webhook "test.slok.dev" denied the request: test message from validator`, sErr.ErrStatus.Message)
						assert.Equal(t, metav1.StatusFailure, sErr.ErrStatus.Status)
					}
				} else {
					// Creation should err, if we are here then we need to clean.
					// nolint: errcheck
					cli.CoreV1().Pods(p.Namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})
				}
			},
		},

		"(invalid, dynamic, core) Having a dynamic webhook, a validating webhook should not allow creating the pod and return a message.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesPod}, []string{version}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
					return &validating.ValidatorResult{
						Valid:   false,
						Message: "test message from validator",
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					ID:        "pod-validating-label",
					Validator: val,
				})
				return vwh
			},
			execTest: func(t *testing.T, cli kubernetes.Interface, _ kubewebhookcrd.Interface) {
				// Crate a pod and check expectations.
				p := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "test", Image: "wrong"}},
					},
				}
				_, err := cli.CoreV1().Pods(p.Namespace).Create(context.TODO(), p, metav1.CreateOptions{})
				if assert.Error(t, err) {
					sErr, ok := err.(*apierrors.StatusError)
					if assert.True(t, ok) {
						assert.Equal(t, `admission webhook "test.slok.dev" denied the request: test message from validator`, sErr.ErrStatus.Message)
						assert.Equal(t, metav1.StatusFailure, sErr.ErrStatus.Status)
					}
				} else {
					// Creation should err, if we are here then we need to clean.
					// nolint: errcheck
					cli.CoreV1().Pods(p.Namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})
				}
			},
		},

		"(valid, static, core) A validating webhook should allow creating the pod.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesPod}, []string{version}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
					return &validating.ValidatorResult{Valid: true}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					ID:        "pod-validating-label",
					Obj:       &corev1.Pod{},
					Validator: val,
				})
				return vwh
			},
			execTest: func(t *testing.T, cli kubernetes.Interface, _ kubewebhookcrd.Interface) {
				// Crate a pod.
				p := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "test", Image: "wrong"}},
					},
				}
				_, err := cli.CoreV1().Pods(p.Namespace).Create(context.TODO(), p, metav1.CreateOptions{})
				require.NoError(t, err)
				// nolint: errcheck
				defer cli.CoreV1().Pods(p.Namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})

				// Check expectations.
				_, err = cli.CoreV1().Pods(p.Namespace).Get(context.TODO(), p.Name, metav1.GetOptions{})
				assert.NoError(t, err, "pod should be present")
			},
		},

		"(invalid, static, CRD) Having a static webhook, a validating webhook should not allow creating the CRD and return a message.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesHouseCRD}, []string{version}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
					h := obj.(*buildingv1.House)
					if h.Spec.Name == "newHouse" {
						return &validating.ValidatorResult{
							Valid:   false,
							Message: "test message from validator",
						}, nil
					}

					return &validating.ValidatorResult{Valid: true}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					ID:        "crd-validating-label",
					Obj:       &buildingv1.House{},
					Validator: val,
				})
				return vwh
			},
			execTest: func(t *testing.T, _ kubernetes.Interface, crdcli kubewebhookcrd.Interface) {
				// Crate a house and check expectations.
				h := &buildingv1.House{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
						Labels: map[string]string{
							"city":      "Bilbo",
							"bathrooms": "2",
						},
					},
					Spec: buildingv1.HouseSpec{
						Name:    "newHouse",
						Address: "whatever 42",
						Owners: []buildingv1.User{
							{Name: "user1", Email: "user1@kubebwehook.slok.dev"},
							{Name: "user2", Email: "user2@kubebwehook.slok.dev"},
						},
					},
				}
				_, err := crdcli.BuildingV1().Houses(h.Namespace).Create(context.TODO(), h, metav1.CreateOptions{})
				if assert.Error(t, err) {
					sErr, ok := err.(*apierrors.StatusError)
					if assert.True(t, ok) {
						assert.Equal(t, `admission webhook "test.slok.dev" denied the request: test message from validator`, sErr.ErrStatus.Message)
						assert.Equal(t, metav1.StatusFailure, sErr.ErrStatus.Status)
					}
				} else {
					// Creation should err, if we are here then we need to clean.
					// nolint: errcheck
					crdcli.BuildingV1().Houses(h.Namespace).Delete(context.TODO(), h.Name, metav1.DeleteOptions{})
				}
			},
		},

		"(invalid, dynamic, CRD) Having a dynamic webhook, a validating webhook should not allow creating the CRD and return a message.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesHouseCRD}, []string{version}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
					labels := obj.GetLabels()

					// We should have the object correctly to return invalid, this wil test we have correctly our object.
					city := labels["city"]
					if city != "Bilbo" {
						return &validating.ValidatorResult{Valid: true}, nil
					}

					return &validating.ValidatorResult{
						Valid:   false,
						Message: "test message from validator",
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					ID:        "crd-validating-label",
					Validator: val,
				})
				return vwh
			},
			execTest: func(t *testing.T, _ kubernetes.Interface, crdcli kubewebhookcrd.Interface) {
				// Crate a house and check expectations.
				h := &buildingv1.House{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
						Labels: map[string]string{
							"city":      "Bilbo",
							"bathrooms": "2",
						},
					},
					Spec: buildingv1.HouseSpec{
						Name:    "newHouse",
						Address: "whatever 42",
						Owners: []buildingv1.User{
							{Name: "user1", Email: "user1@kubebwehook.slok.dev"},
							{Name: "user2", Email: "user2@kubebwehook.slok.dev"},
						},
					},
				}
				_, err := crdcli.BuildingV1().Houses(h.Namespace).Create(context.TODO(), h, metav1.CreateOptions{})
				if assert.Error(t, err) {
					sErr, ok := err.(*apierrors.StatusError)
					if assert.True(t, ok) {
						assert.Equal(t, `admission webhook "test.slok.dev" denied the request: test message from validator`, sErr.ErrStatus.Message)
						assert.Equal(t, metav1.StatusFailure, sErr.ErrStatus.Status)
					}
				} else {
					// Creation should err, if we are here then we need to clean.
					// nolint: errcheck
					crdcli.BuildingV1().Houses(h.Namespace).Delete(context.TODO(), h.Name, metav1.DeleteOptions{})
				}
			},
		},

		"(invalid, static, unstructured, CRD) Having a static webhook forcing unstructured, a validating webhook should not allow creating the CRD and return a message.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesHouseCRD}, []string{version}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
					labels := obj.GetLabels()

					// We should have the object correctly to return invalid, this wil test we have correctly our object.
					city := labels["city"]
					if city != "Bilbo" {
						return &validating.ValidatorResult{Valid: true}, nil
					}

					return &validating.ValidatorResult{
						Valid:   false,
						Message: "test message from validator",
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					ID:        "crd-validating-label",
					Obj:       &unstructured.Unstructured{},
					Validator: val,
				})
				return vwh
			},
			execTest: func(t *testing.T, _ kubernetes.Interface, crdcli kubewebhookcrd.Interface) {
				// Crate a house and check expectations.
				h := &buildingv1.House{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
						Labels: map[string]string{
							"city":      "Bilbo",
							"bathrooms": "2",
						},
					},
					Spec: buildingv1.HouseSpec{
						Name:    "newHouse",
						Address: "whatever 42",
						Owners: []buildingv1.User{
							{Name: "user1", Email: "user1@kubebwehook.slok.dev"},
							{Name: "user2", Email: "user2@kubebwehook.slok.dev"},
						},
					},
				}
				_, err := crdcli.BuildingV1().Houses(h.Namespace).Create(context.TODO(), h, metav1.CreateOptions{})
				if assert.Error(t, err) {
					sErr, ok := err.(*apierrors.StatusError)
					if assert.True(t, ok) {
						assert.Equal(t, `admission webhook "test.slok.dev" denied the request: test message from validator`, sErr.ErrStatus.Message)
						assert.Equal(t, metav1.StatusFailure, sErr.ErrStatus.Status)
					}
				} else {
					// Creation should err, if we are here then we need to clean.
					// nolint: errcheck
					crdcli.BuildingV1().Houses(h.Namespace).Delete(context.TODO(), h.Name, metav1.DeleteOptions{})
				}
			},
		},

		"(valid, static, CRD) Having a static webhook, a validating webhook should allow creating the CRD and return a message.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesHouseCRD}, []string{version}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
					h := obj.(*buildingv1.House)
					if h.Spec.Name == "newHouse" {
						return &validating.ValidatorResult{StopChain: true, Valid: true}, nil
					}

					return &validating.ValidatorResult{Valid: false}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					ID:        "crd-validating-label",
					Obj:       &buildingv1.House{},
					Validator: val,
				})
				return vwh
			},
			execTest: func(t *testing.T, _ kubernetes.Interface, crdcli kubewebhookcrd.Interface) {
				// Create a house and check expectations.
				h := &buildingv1.House{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
						Labels: map[string]string{
							"city":      "Bilbo",
							"bathrooms": "2",
						},
					},
					Spec: buildingv1.HouseSpec{
						Name:    "newHouse",
						Address: "whatever 42",
						Owners: []buildingv1.User{
							{Name: "user1", Email: "user1@kubebwehook.slok.dev"},
							{Name: "user2", Email: "user2@kubebwehook.slok.dev"},
						},
					},
				}
				_, err := crdcli.BuildingV1().Houses(h.Namespace).Create(context.TODO(), h, metav1.CreateOptions{})
				require.NoError(t, err)
				// nolint: errcheck
				defer crdcli.BuildingV1().Houses(h.Namespace).Delete(context.TODO(), h.Name, metav1.DeleteOptions{})

				// Check expectations.
				_, err = crdcli.BuildingV1().Houses(h.Namespace).Get(context.TODO(), h.Name, metav1.GetOptions{})
				assert.NoError(t, err, "house should be present")
			},
		},

		"(valid, static, core, delete) Having a static webhook, a validating webhook should allow deleting the pod.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesDeletePod}, []string{version}),
			webhook: func() webhook.Webhook {
				val := validating.ValidatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
					// Allow if it has our label.
					if l := obj.GetLabels()["kubewebhook"]; l == "test" {
						return &validating.ValidatorResult{StopChain: true, Valid: true}, nil
					}

					return &validating.ValidatorResult{
						Valid:   false,
						Message: "test message from validator",
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					ID:        "pod-validating-delete",
					Obj:       &corev1.Pod{},
					Validator: val,
				})
				return vwh
			},
			execTest: func(t *testing.T, cli kubernetes.Interface, _ kubewebhookcrd.Interface) {
				assert := assert.New(t)
				require := require.New(t)

				p := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
						Labels:    map[string]string{"kubewebhook": "test"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "test", Image: "wrong"}},
					},
				}
				_, err := cli.CoreV1().Pods(p.Namespace).Create(context.TODO(), p, metav1.CreateOptions{})
				require.NoError(err)

				err = cli.CoreV1().Pods(p.Namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})
				require.NoError(err)

				// Give time so deleting takes place.
				time.Sleep(15 * time.Second)

				// Check expectations.
				_, err = cli.CoreV1().Pods(p.Namespace).Get(context.TODO(), p.Name, metav1.GetOptions{})
				assert.Error(err, "pod shouldn't be present")
			},
		},

		"(valid, dynamic, CRD, delete) Having a dynamic webhook, a validating webhook should allow deleting the CRD.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesDeletePod}, []string{version}),
			webhook: func() webhook.Webhook {
				val := validating.ValidatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
					// Allow if it has our label.
					if l := obj.GetLabels()["city"]; l == "Bilbo" {
						return &validating.ValidatorResult{StopChain: true, Valid: true}, nil
					}

					return &validating.ValidatorResult{
						Valid:   false,
						Message: "test message from validator",
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					ID:        "pod-dynamic-validating-delete",
					Validator: val,
				})
				return vwh
			},
			execTest: func(t *testing.T, _ kubernetes.Interface, crdcli kubewebhookcrd.Interface) {
				assert := assert.New(t)
				require := require.New(t)

				h := &buildingv1.House{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
						Labels:    map[string]string{"city": "Bilbo"},
					},
					Spec: buildingv1.HouseSpec{Name: "newHouse"},
				}

				_, err := crdcli.BuildingV1().Houses(h.Namespace).Create(context.TODO(), h, metav1.CreateOptions{})
				require.NoError(err)

				err = crdcli.BuildingV1().Houses(h.Namespace).Delete(context.TODO(), h.Name, metav1.DeleteOptions{})
				require.NoError(err)

				// Give time so deleting takes place.
				time.Sleep(10 * time.Second)

				// Check expectations.
				_, err = cli.CoreV1().Pods(h.Namespace).Get(context.TODO(), h.Name, metav1.GetOptions{})
				assert.Error(err, "house shouldn't be present")
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Register webhooks.
			_, err := cli.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), test.webhookRegisterCfg, metav1.CreateOptions{})
			if err != nil {
				assert.FailNow(t, "error registering webhooks kubernetes client", err.Error())
			}
			// nolint: errcheck
			defer cli.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.TODO(), test.webhookRegisterCfg.Name, metav1.DeleteOptions{})

			// Start validating webhook server.
			wh := test.webhook()
			h := whhttp.MustHandlerFor(wh)
			srv := http.Server{
				Handler: h,
				Addr:    cfg.ListenAddress,
			}
			go func() {
				err := srv.ListenAndServeTLS(cfg.WebhookCertPath, cfg.WebhookCertKeyPath)
				if err != nil && err != http.ErrServerClosed {
					assert.FailNow(t, "error serving webhook", err.Error())
				}
			}()
			// nolint: errcheck
			defer srv.Shutdown(context.TODO())

			// Wait a bit to get ready with the webhook server goroutine.
			time.Sleep(2 * time.Second)

			// Execute the tests.
			test.execTest(t, cli, crdcli)
		})
	}
}

func testValidatingWebhookWarnings(t *testing.T) {
	cfg := helperconfig.GetTestEnvConfig(t)

	tests := map[string]struct {
		webhookRegisterCfg *arv1.ValidatingWebhookConfiguration
		webhook            func() webhook.Webhook
		execTest           func(t *testing.T, cli kubernetes.Interface, crdcli kubewebhookcrd.Interface)
		expWarnings        string
	}{
		"Warning messages should be received by a validating webhook that accepts.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesPod}, []string{"v1"}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
					return &validating.ValidatorResult{
						Valid: true,
						Warnings: []string{
							"this is the first warning",
							"and this is the second warning",
						},
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					ID:        "pod-validating-label",
					Obj:       &corev1.Pod{},
					Validator: val,
				})
				return vwh
			},
			execTest: func(t *testing.T, cli kubernetes.Interface, _ kubewebhookcrd.Interface) {
				// Create a pod.
				p := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
					},
					Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "test", Image: "wrong", Ports: []corev1.ContainerPort{{ContainerPort: 8080}}}}},
				}
				_, err := cli.CoreV1().Pods(p.Namespace).Create(context.TODO(), p, metav1.CreateOptions{})
				require.NoError(t, err)
				// nolint: errcheck
				defer cli.CoreV1().Pods(p.Namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})
			},
			expWarnings: "Warning: this is the first warning\nWarning: and this is the second warning\n",
		},

		"Warning messages should be received by a validating webhook that does not accept.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesPod}, []string{"v1"}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*validating.ValidatorResult, error) {
					return &validating.ValidatorResult{
						Valid:   false,
						Message: "test message from validator",
						Warnings: []string{
							"this is the first warning",
							"and this is the second warning",
						},
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					ID:        "pod-validating-label",
					Obj:       &corev1.Pod{},
					Validator: val,
				})
				return vwh
			},
			execTest: func(t *testing.T, cli kubernetes.Interface, _ kubewebhookcrd.Interface) {
				// Create a pod.
				p := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
					},
					Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "test", Image: "wrong", Ports: []corev1.ContainerPort{{ContainerPort: 8080}}}}},
				}

				_, err := cli.CoreV1().Pods(p.Namespace).Create(context.TODO(), p, metav1.CreateOptions{})
				if assert.Error(t, err) {
					sErr, ok := err.(*apierrors.StatusError)
					if assert.True(t, ok) {
						assert.Equal(t, `admission webhook "test.slok.dev" denied the request: test message from validator`, sErr.ErrStatus.Message)
						assert.Equal(t, metav1.StatusFailure, sErr.ErrStatus.Status)
					}
				} else {
					// Creation should err, if we are here then we need to clean.
					// nolint: errcheck
					cli.CoreV1().Pods(p.Namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})
				}
			},
			expWarnings: "Warning: this is the first warning\nWarning: and this is the second warning\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var gotWarnings bytes.Buffer

			cli, err := helpercli.GetK8sSTDClients(cfg.KubeConfigPath, &gotWarnings)
			require.NoError(t, err, "error getting kubernetes client")
			crdcli, err := helpercli.GetK8sCRDClients(cfg.KubeConfigPath, &gotWarnings)
			require.NoError(t, err, "error getting kubernetes CRD client")

			// Register webhooks.
			_, err = cli.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), test.webhookRegisterCfg, metav1.CreateOptions{})
			require.NoError(t, err, "error registering webhooks kubernetes client")
			// nolint: errcheck
			defer cli.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.TODO(), test.webhookRegisterCfg.Name, metav1.DeleteOptions{})

			// Start validating webhook server.
			wh := test.webhook()
			h := whhttp.MustHandlerFor(wh)
			srv := http.Server{
				Handler: h,
				Addr:    cfg.ListenAddress,
			}
			go func() {
				err := srv.ListenAndServeTLS(cfg.WebhookCertPath, cfg.WebhookCertKeyPath)
				if err != nil && err != http.ErrServerClosed {
					assert.FailNow(t, "error serving webhook", err.Error())
				}
			}()
			// nolint: errcheck
			defer srv.Shutdown(context.TODO())

			// Wait a bit to get ready with the webhook server goroutine.
			time.Sleep(2 * time.Second)

			// Execute test.
			test.execTest(t, cli, crdcli)

			// Check warnings.
			assert.Equal(t, test.expWarnings, gotWarnings.String())
		})
	}
}

func TestValidatingWebhookV1Beta1(t *testing.T) {
	testValidatingWebhookCommon(t, "v1beta1")
}

func TestValidatingWebhookV1(t *testing.T) {
	testValidatingWebhookCommon(t, "v1")
	testValidatingWebhookWarnings(t)
}
