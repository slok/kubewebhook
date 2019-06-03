// +build integration

package webhook_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	arv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	whhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/validating"
	helpercli "github.com/slok/kubewebhook/test/integration/helper/cli"
	helperconfig "github.com/slok/kubewebhook/test/integration/helper/config"
)

func getValidatingWebhookConfig(t *testing.T, cfg helperconfig.TestEnvConfig, rules []arv1beta1.RuleWithOperations) *arv1beta1.ValidatingWebhookConfiguration {
	return &arv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "integration-test-webhook",
		},
		Webhooks: []arv1beta1.Webhook{
			{
				Name: "test.slok.dev",
				ClientConfig: arv1beta1.WebhookClientConfig{
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
	// using a local k3s + serveo stack (check /test/integration/helper/config).
	//cfg = helperconfig.GetTestDevelopmentEnvConfig(t)

	cli, err := helpercli.GetK8sClients(cfg.KubeConfigPath)
	require.NoError(t, err, "error getting kubernetes client")

	tests := map[string]struct {
		webhookRegisterCfg *arv1beta1.ValidatingWebhookConfiguration
		webhook            func() webhook.Webhook
		execTest           func(t *testing.T, cli kubernetes.Interface)
	}{
		"A validating webhook should not allow creating the pod and return a message.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1beta1.RuleWithOperations{webhookRulesPod}),
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
			execTest: func(t *testing.T, cli kubernetes.Interface) {
				// Crate a pod and check expectations.
				p := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{corev1.Container{Name: "test", Image: "wrong"}},
					},
				}
				_, err := cli.CoreV1().Pods("default").Create(p)
				if assert.Error(t, err) {
					sErr, ok := err.(*apierrors.StatusError)
					if assert.True(t, ok) {
						assert.Equal(t, `admission webhook "test.slok.dev" denied the request: test message from validator`, sErr.ErrStatus.Message)
						assert.Equal(t, metav1.StatusFailure, sErr.ErrStatus.Status)
					}
				} else {
					// Creation should err, if we are here then we need to clean.
					cli.CoreV1().Pods("default").Delete(p.Name, &metav1.DeleteOptions{})
				}
			},
		},

		"A validating webhook should allow creating the pod.": {
			webhookRegisterCfg: getValidatingWebhookConfig(t, cfg, []arv1beta1.RuleWithOperations{webhookRulesPod}),
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
			execTest: func(t *testing.T, cli kubernetes.Interface) {
				// Crate a pod.
				p := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test2",
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{corev1.Container{Name: "test", Image: "wrong"}},
					},
				}
				_, err := cli.CoreV1().Pods("default").Create(p)
				require.NoError(t, err)
				defer cli.CoreV1().Pods("default").Delete(p.Name, &metav1.DeleteOptions{})

				// Check expectations.
				_, err = cli.CoreV1().Pods("default").Get("test2", metav1.GetOptions{})
				assert.NoError(t, err, "pod should be present")
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Register webhooks.
			_, err := cli.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(test.webhookRegisterCfg)
			if err != nil {
				assert.FailNow(t, "error registering webhooks kubernetes client", err.Error())
			}
			defer cli.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete(test.webhookRegisterCfg.Name, &metav1.DeleteOptions{})

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

			// Execute the tests.
			test.execTest(t, cli)
		})
	}
}
