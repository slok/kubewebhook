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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	whhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/webhook"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
	helpercli "github.com/slok/kubewebhook/test/integration/helper/cli"
	helperconfig "github.com/slok/kubewebhook/test/integration/helper/config"
)

func getMutatingWebhookConfig(t *testing.T, cfg helperconfig.TestEnvConfig, rules []arv1beta1.RuleWithOperations) *arv1beta1.MutatingWebhookConfiguration {
	return &arv1beta1.MutatingWebhookConfiguration{
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

var (
	webhookRulesPod = arv1beta1.RuleWithOperations{
		Operations: []arv1beta1.OperationType{"CREATE"},
		Rule: arv1beta1.Rule{
			APIGroups:   []string{""},
			APIVersions: []string{"v1"},
			Resources:   []string{"pods"},
		},
	}
)

func TestMutatingWebhook(t *testing.T) {
	cfg := helperconfig.GetTestEnvConfig(t)
	// Use this configuration if you are developing the tests and you are
	// using a local k3s + serveo stack (check /test/integration/helper/config).
	//cfg = helperconfig.GetTestDevelopmentEnvConfig(t)

	cli, err := helpercli.GetK8sClients(cfg.KubeConfigPath)
	require.NoError(t, err, "error getting kubernetes client")

	tests := map[string]struct {
		webhookRegisterCfg *arv1beta1.MutatingWebhookConfiguration
		webhook            func() webhook.Webhook
		execTest           func(t *testing.T, cli kubernetes.Interface)
	}{
		"A mutation on a pod creation should mutate the pod labels, rewrite the existing ones, and add the missing ones.": {
			webhookRegisterCfg: getMutatingWebhookConfig(t, cfg, []arv1beta1.RuleWithOperations{webhookRulesPod}),
			webhook: func() webhook.Webhook {
				// Our mutator logic.
				mut := mutating.MutatorFunc(func(ctx context.Context, obj metav1.Object) (bool, error) {
					pod := obj.(*corev1.Pod)
					if pod.Labels == nil {
						pod.Labels = map[string]string{}
					}
					pod.Labels["name"] = "Bruce"
					pod.Labels["lastName"] = "Wayne"
					pod.Labels["nickname"] = "Batman"
					return false, nil
				})
				mwh, _ := mutating.NewWebhook(mutating.WebhookConfig{
					Name: "pod-mutator-label",
					Obj:  &corev1.Pod{},
				}, mut, nil, nil, nil)
				return mwh
			},
			execTest: func(t *testing.T, cli kubernetes.Interface) {
				// Try creating a pod.
				p := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							"nickname": "Dark-knight",
							"city":     "Gotham",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "test",
								Image: "wrong",
							},
						},
					},
				}
				_, err := cli.CoreV1().Pods("default").Create(p)
				require.NoError(t, err)
				defer cli.CoreV1().Pods("default").Delete(p.Name, &metav1.DeleteOptions{})

				// Check expectations.
				expLabels := map[string]string{
					"name":     "Bruce",
					"lastName": "Wayne",
					"nickname": "Batman",
					"city":     "Gotham",
				}
				pod, err := cli.CoreV1().Pods("default").Get("test", metav1.GetOptions{})
				if assert.NoError(t, err) {
					assert.Equal(t, expLabels, pod.Labels)
				}
			},
		},

		"A mutation on a pod creation should mutate the containers, mutate one of them and add a new one.": {
			webhookRegisterCfg: getMutatingWebhookConfig(t, cfg, []arv1beta1.RuleWithOperations{webhookRulesPod}),
			webhook: func() webhook.Webhook {
				// Our mutator logic.
				mut := mutating.MutatorFunc(func(ctx context.Context, obj metav1.Object) (bool, error) {
					pod := obj.(*corev1.Pod)
					// Add a container.
					pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: "test3", Image: "wrong3"})

					// Edit a container.
					pod.Spec.Containers[0].Ports[1].ContainerPort = 7071

					// Sort containers,
					c0 := pod.Spec.Containers[0]
					c1 := pod.Spec.Containers[1]
					c2 := pod.Spec.Containers[2]
					pod.Spec.Containers[0] = c1
					pod.Spec.Containers[1] = c2
					pod.Spec.Containers[2] = c0

					return false, nil
				})
				mwh, _ := mutating.NewWebhook(mutating.WebhookConfig{
					Name: "pod-mutator-test2",
					Obj:  &corev1.Pod{},
				}, mut, nil, nil, nil)
				return mwh
			},
			execTest: func(t *testing.T, cli kubernetes.Interface) {
				// Create a pod.
				p := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test2",
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "test",
								Image: "wrong",
								Ports: []corev1.ContainerPort{
									{ContainerPort: 8080},
									{ContainerPort: 8081},
									{ContainerPort: 8082},
								},
							},
							corev1.Container{Name: "test2", Image: "wrong2"},
						},
					},
				}
				_, err := cli.CoreV1().Pods("default").Create(p)
				require.NoError(t, err)
				defer cli.CoreV1().Pods("default").Delete(p.Name, &metav1.DeleteOptions{})

				// Check expectations.
				expContainers := []corev1.Container{
					corev1.Container{Name: "test2", Image: "wrong2"},
					corev1.Container{Name: "test3", Image: "wrong3"},
					corev1.Container{
						Name:  "test",
						Image: "wrong",
						Ports: []corev1.ContainerPort{
							{ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
							{ContainerPort: 7071, Protocol: corev1.ProtocolTCP},
							{ContainerPort: 8082, Protocol: corev1.ProtocolTCP},
						},
					},
				}
				pod, err := cli.CoreV1().Pods("default").Get("test2", metav1.GetOptions{})
				if assert.NoError(t, err) {
					// Sanitize default settings on containers before checking expectations.
					for i, container := range pod.Spec.Containers {
						container.VolumeMounts = nil
						container.ImagePullPolicy = ""
						container.TerminationMessagePath = ""
						container.TerminationMessagePolicy = ""
						pod.Spec.Containers[i] = container
					}
					assert.Equal(t, expContainers, pod.Spec.Containers)
				}
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Register webhooks.
			_, err := cli.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(test.webhookRegisterCfg)
			require.NoError(t, err, "error registering webhooks kubernetes client")
			defer cli.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(test.webhookRegisterCfg.Name, &metav1.DeleteOptions{})

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

			// Execute test.
			test.execTest(t, cli)
		})
	}
}
