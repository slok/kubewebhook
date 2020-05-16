// +build integration

package webhook_test

import (
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
	"k8s.io/client-go/kubernetes"

	whhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/validating"
	buildingv1 "github.com/slok/kubewebhook/test/integration/crd/apis/building/v1"
	kubewebhookcrd "github.com/slok/kubewebhook/test/integration/crd/client/clientset/versioned"
	helpercli "github.com/slok/kubewebhook/test/integration/helper/cli"
	helperconfig "github.com/slok/kubewebhook/test/integration/helper/config"
)

func getValidatingWebhookConfig(t *testing.T, cfg helperconfig.TestEnvConfig, rules []arv1.RuleWithOperations) *arv1.ValidatingWebhookConfiguration {
	whSideEffect := arv1.SideEffectClassNone
	whFailurePolicy := arv1.Fail
	var timeoutSecs int32 = 30
	return &arv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "integration-test-webhook",
		},
		Webhooks: []arv1.ValidatingWebhook{
			{
				Name:                    "test.slok.dev",
				AdmissionReviewVersions: []string{"v1beta1"},
				FailurePolicy:           &whFailurePolicy,
				TimeoutSeconds:          &timeoutSecs,
				SideEffects:             &whSideEffect,
				ClientConfig: arv1.WebhookClientConfig{
					URL:      &cfg.WebhookURL,
					CABundle: []byte(cfg.WebhookCert),
				},
				Rules: rules,
			},
		},
	}
}

func TestValidatingWebhook(t *testing.T) {
	cfg := helperconfig.GetTestEnvConfig(t)
	// Use this configuration if you are developing the tests and you are
	// using a local k3s + ngrok stack (check /test/integration/helper/config).
	//cfg = helperconfig.GetTestDevelopmentEnvConfig(t)

	cli, err := helpercli.GetK8sSTDClients(cfg.KubeConfigPath)
	require.NoError(t, err, "error getting kubernetes client")
	crdcli, err := helpercli.GetK8sCRDClients(cfg.KubeConfigPath)
	require.NoError(t, err, "error getting kubernetes CRD client")

	tests := map[string]struct {
		webhookRegisterCfg *arv1.ValidatingWebhookConfiguration
		webhook            func() webhook.Webhook
		execTest           func(t *testing.T, cli kubernetes.Interface, crdcli kubewebhookcrd.Interface)
	}{
		"Having a static webhook, a validating webhook should not allow creating the pod and return a message.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesPod}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
					return true, validating.ValidatorResult{
						Valid:   false,
						Message: "test message from validator",
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					Name: "pod-validating-label",
					Obj:  &corev1.Pod{},
				}, val, nil, nil, nil)
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
					cli.CoreV1().Pods(p.Namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})
				}
			},
		},

		"Having a dynamic webhook, a validating webhook should not allow creating the pod and return a message.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesPod}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
					return true, validating.ValidatorResult{
						Valid:   false,
						Message: "test message from validator",
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{Name: "pod-validating-label"}, val, nil, nil, nil)
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
					cli.CoreV1().Pods(p.Namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})
				}
			},
		},

		"Having a static webhook, a validating webhook should allow creating the pod.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesPod}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
					return true, validating.ValidatorResult{Valid: true}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					Name: "pod-validating-label",
					Obj:  &corev1.Pod{},
				}, val, nil, nil, nil)
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
				defer cli.CoreV1().Pods(p.Namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})

				// Check expectations.
				_, err = cli.CoreV1().Pods(p.Namespace).Get(context.TODO(), p.Name, metav1.GetOptions{})
				assert.NoError(t, err, "pod should be present")
			},
		},

		"Having a static webhook, a validating webhook should not allow creating the CRD and return a message.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesHouseCRD}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
					h := obj.(*buildingv1.House)
					if h.Spec.Name == "newHouse" {
						return true, validating.ValidatorResult{
							Valid:   false,
							Message: "test message from validator",
						}, nil
					}

					return true, validating.ValidatorResult{Valid: true}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					Name: "crd-validating-label",
					Obj:  &buildingv1.House{},
				}, val, nil, nil, nil)
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
					crdcli.BuildingV1().Houses(h.Namespace).Delete(context.TODO(), h.Name, metav1.DeleteOptions{})
				}
			},
		},

		"Having a dynamic webhook, a validating webhook should not allow creating the CRD and return a message.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesHouseCRD}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
					return true, validating.ValidatorResult{
						Valid:   false,
						Message: "test message from validator",
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{Name: "crd-validating-label"}, val, nil, nil, nil)
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
					crdcli.BuildingV1().Houses(h.Namespace).Delete(context.TODO(), h.Name, metav1.DeleteOptions{})
				}
			},
		},

		"Having a static webhook, a validating webhook should allow creating the CRD and return a message.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesHouseCRD}),
			webhook: func() webhook.Webhook {
				// Our validator logic.
				val := validating.ValidatorFunc(func(ctx context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
					h := obj.(*buildingv1.House)
					if h.Spec.Name == "newHouse" {
						return true, validating.ValidatorResult{Valid: true}, nil
					}

					return true, validating.ValidatorResult{Valid: false}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					Name: "crd-validating-label",
					Obj:  &buildingv1.House{},
				}, val, nil, nil, nil)
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
				defer crdcli.BuildingV1().Houses(h.Namespace).Delete(context.TODO(), h.Name, metav1.DeleteOptions{})

				// Check expectations.
				_, err = crdcli.BuildingV1().Houses(h.Namespace).Get(context.TODO(), h.Name, metav1.GetOptions{})
				assert.NoError(t, err, "house should be present")
			},
		},

		"Having a static webhook, a validating webhook should allow deleting the pod.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesDeletePod}),
			webhook: func() webhook.Webhook {
				val := validating.ValidatorFunc(func(ctx context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
					// Allow if it has our label.
					if l := obj.GetLabels()["kubewebhook"]; l == "test" {
						return true, validating.ValidatorResult{Valid: true}, nil
					}

					return true, validating.ValidatorResult{
						Valid:   false,
						Message: "test message from validator",
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					Name: "pod-validating-delete",
					Obj:  &corev1.Pod{},
				}, val, nil, nil, nil)
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
				time.Sleep(5 * time.Second)

				// Check expectations.
				_, err = cli.CoreV1().Pods(p.Namespace).Get(context.TODO(), p.Name, metav1.GetOptions{})
				assert.Error(err, "pod shouldn't be present")
			},
		},

		"Having a dynamic webhook, a validating webhook should allow deleting the CRD.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesDeletePod}),
			webhook: func() webhook.Webhook {
				val := validating.ValidatorFunc(func(ctx context.Context, obj metav1.Object) (bool, validating.ValidatorResult, error) {
					// Allow if it has our label.
					if l := obj.GetLabels()["city"]; l == "Bilbo" {
						return true, validating.ValidatorResult{Valid: true}, nil
					}

					return true, validating.ValidatorResult{
						Valid:   false,
						Message: "test message from validator",
					}, nil
				})
				vwh, _ := validating.NewWebhook(validating.WebhookConfig{
					Name: "pod-dynamic-validating-delete",
				}, val, nil, nil, nil)
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
				time.Sleep(5 * time.Second)

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
			defer cli.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.TODO(), test.webhookRegisterCfg.Name, metav1.DeleteOptions{})

			// Start mutating webhook server.
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
			defer srv.Shutdown(context.TODO())

			// Wait a bit to get ready with the webhook server goroutine.
			time.Sleep(2 * time.Second)

			// Execute the tests.
			test.execTest(t, cli, crdcli)
		})
	}
}
