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
	var timeoutSecs int32 = 30
	return &arv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "integration-test-webhook",
		},
		Webhooks: []arv1.ValidatingWebhook{
			{
				Name:                    "test.slok.dev",
				AdmissionReviewVersions: []string{"v1beta1"},
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
		"A validating webhook should not allow creating the pod and return a message.": {
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
						Containers: []corev1.Container{corev1.Container{Name: "test", Image: "wrong"}},
					},
				}
				_, err := cli.CoreV1().Pods(p.Namespace).Create(p)
				if assert.Error(t, err) {
					sErr, ok := err.(*apierrors.StatusError)
					if assert.True(t, ok) {
						assert.Equal(t, `admission webhook "test.slok.dev" denied the request: test message from validator`, sErr.ErrStatus.Message)
						assert.Equal(t, metav1.StatusFailure, sErr.ErrStatus.Status)
					}
				} else {
					// Creation should err, if we are here then we need to clean.
					cli.CoreV1().Pods(p.Namespace).Delete(p.Name, &metav1.DeleteOptions{})
				}
			},
		},

		"A validating webhook should allow creating the pod.": {
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
						Containers: []corev1.Container{corev1.Container{Name: "test", Image: "wrong"}},
					},
				}
				_, err := cli.CoreV1().Pods(p.Namespace).Create(p)
				require.NoError(t, err)
				defer cli.CoreV1().Pods(p.Namespace).Delete(p.Name, &metav1.DeleteOptions{})

				// Check expectations.
				_, err = cli.CoreV1().Pods(p.Namespace).Get(p.Name, metav1.GetOptions{})
				assert.NoError(t, err, "pod should be present")
			},
		},

		"A validating webhook should not allow creating the CRD and return a message.": {
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
				_, err := crdcli.BuildingV1().Houses(h.Namespace).Create(h)
				if assert.Error(t, err) {
					sErr, ok := err.(*apierrors.StatusError)
					if assert.True(t, ok) {
						assert.Equal(t, `admission webhook "test.slok.dev" denied the request: test message from validator`, sErr.ErrStatus.Message)
						assert.Equal(t, metav1.StatusFailure, sErr.ErrStatus.Status)
					}
				} else {
					// Creation should err, if we are here then we need to clean.
					crdcli.BuildingV1().Houses(h.Namespace).Delete(h.Name, &metav1.DeleteOptions{})
				}
			},
		},

		"A validating webhook should allow creating the CRD and return a message.": {
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
				_, err := crdcli.BuildingV1().Houses(h.Namespace).Create(h)
				require.NoError(t, err)
				defer crdcli.BuildingV1().Houses(h.Namespace).Delete(h.Name, &metav1.DeleteOptions{})

				// Check expectations.
				_, err = crdcli.BuildingV1().Houses(h.Namespace).Get(h.Name, metav1.GetOptions{})
				assert.NoError(t, err, "house should be present")
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Register webhooks.
			_, err := cli.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(test.webhookRegisterCfg)
			if err != nil {
				assert.FailNow(t, "error registering webhooks kubernetes client", err.Error())
			}
			defer cli.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(test.webhookRegisterCfg.Name, &metav1.DeleteOptions{})

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
