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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"

	whhttp "github.com/slok/kubewebhook/v2/pkg/http"
	"github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/webhook"
	"github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	buildingv1 "github.com/slok/kubewebhook/v2/test/integration/crd/apis/building/v1"
	kubewebhookcrd "github.com/slok/kubewebhook/v2/test/integration/crd/client/clientset/versioned"
	helpercli "github.com/slok/kubewebhook/v2/test/integration/helper/cli"
	helperconfig "github.com/slok/kubewebhook/v2/test/integration/helper/config"
)

// testMutatingWebhookCommon tests the common use cases that should be shared among all webhook versions
// so the version of the webhook (v1 or v1beta1) should not make a difference.
func testMutatingWebhookCommon(t *testing.T, version string) {
	var (
		trueBool  = true
		falseBool = false
	)

	cfg := helperconfig.GetTestEnvConfig(t)
	cli, err := helpercli.GetK8sSTDClients(cfg.KubeConfigPath, nil)
	require.NoError(t, err, "error getting kubernetes client")
	crdcli, err := helpercli.GetK8sCRDClients(cfg.KubeConfigPath, nil)
	require.NoError(t, err, "error getting kubernetes CRD client")

	tests := map[string]struct {
		webhookRegisterCfg *arv1.MutatingWebhookConfiguration
		webhook            func() webhook.Webhook
		execTest           func(t *testing.T, cli kubernetes.Interface, crdcli kubewebhookcrd.Interface)
	}{
		"(static, core) Having a static webhook, a mutation on a pod creation should mutate the containers, mutate one of them and add a new one.": {
			webhookRegisterCfg: getMutatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesPod}, []string{version}),
			webhook: func() webhook.Webhook {
				// Our mutator logic.
				mut := mutating.MutatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*mutating.MutatorResult, error) {
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

					return &mutating.MutatorResult{MutatedObject: pod}, nil
				})
				mwh, _ := mutating.NewWebhook(mutating.WebhookConfig{
					ID:      "pod-mutator-test2",
					Obj:     &corev1.Pod{},
					Mutator: mut,
				})
				return mwh
			},
			execTest: func(t *testing.T, cli kubernetes.Interface, _ kubewebhookcrd.Interface) {
				// Create a pod.
				p := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test",
								Image: "wrong",
								Ports: []corev1.ContainerPort{
									{ContainerPort: 8080},
									{ContainerPort: 8081},
									{ContainerPort: 8082},
								},
							},
							{Name: "test2", Image: "wrong2"},
						},
					},
				}
				_, err := cli.CoreV1().Pods(p.Namespace).Create(context.TODO(), p, metav1.CreateOptions{})
				require.NoError(t, err)
				// nolint: errcheck
				defer cli.CoreV1().Pods(p.Namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})

				// Check expectations.
				expContainers := []corev1.Container{
					{Name: "test2", Image: "wrong2"},
					{Name: "test3", Image: "wrong3"},
					{
						Name:  "test",
						Image: "wrong",
						Ports: []corev1.ContainerPort{
							{ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
							{ContainerPort: 7071, Protocol: corev1.ProtocolTCP},
							{ContainerPort: 8082, Protocol: corev1.ProtocolTCP},
						},
					},
				}
				pod, err := cli.CoreV1().Pods(p.Namespace).Get(context.TODO(), p.Name, metav1.GetOptions{})
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

		"(dynamic, core) Having a dynamic webhook, a mutation on a pod creation should mutate the pod labels, rewrite the existing ones, and add the missing ones.": {
			webhookRegisterCfg: getMutatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesPod}, []string{version}),
			webhook: func() webhook.Webhook {
				// Our mutator logic.
				mut := mutating.MutatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*mutating.MutatorResult, error) {
					pod := obj.(*corev1.Pod)

					if pod.Labels == nil {
						pod.Labels = map[string]string{}
					}
					pod.Labels["name"] = "Bruce"
					pod.Labels["lastName"] = "Wayne"
					pod.Labels["nickname"] = "Batman"

					return &mutating.MutatorResult{MutatedObject: pod}, nil
				})
				mwh, _ := mutating.NewWebhook(mutating.WebhookConfig{
					ID:      "pod-mutator-label",
					Mutator: mut,
				})
				return mwh
			},
			execTest: func(t *testing.T, cli kubernetes.Interface, _ kubewebhookcrd.Interface) {
				// Try creating a pod.
				p := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
						Labels: map[string]string{
							"nickname": "Dark-knight",
							"city":     "Gotham",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test",
								Image: "wrong",
							},
						},
					},
				}
				_, err := cli.CoreV1().Pods(p.Namespace).Create(context.TODO(), p, metav1.CreateOptions{})
				require.NoError(t, err)
				// nolint: errcheck
				defer cli.CoreV1().Pods(p.Namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})

				// Check expectations.
				expLabels := map[string]string{
					"name":     "Bruce",
					"lastName": "Wayne",
					"nickname": "Batman",
					"city":     "Gotham",
				}
				pod, err := cli.CoreV1().Pods(p.Namespace).Get(context.TODO(), p.Name, metav1.GetOptions{})
				if assert.NoError(t, err) {
					assert.Equal(t, expLabels, pod.Labels)
				}
			},
		},

		"(static, CRD) Having a static webhook, a mutation on a CRD creation should mutate the the CRD fields, rewrite the existing ones, and add the missing ones.": {
			webhookRegisterCfg: getMutatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesHouseCRD}, []string{version}),
			webhook: func() webhook.Webhook {
				// Our mutator logic.
				mut := mutating.MutatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*mutating.MutatorResult, error) {
					house := obj.(*buildingv1.House)
					house.Spec.Name = "changed-name"
					house.Spec.Active = &trueBool
					house.Spec.Address = ""
					house.Spec.Owners = []buildingv1.User{
						{Name: "user1", Email: "user1@kubebwehook.slok.dev"},
					}
					return &mutating.MutatorResult{MutatedObject: house}, nil
				})
				mwh, _ := mutating.NewWebhook(mutating.WebhookConfig{
					ID:      "house-mutator-label",
					Obj:     &buildingv1.House{},
					Mutator: mut,
				})
				return mwh
			},
			execTest: func(t *testing.T, _ kubernetes.Interface, crdcli kubewebhookcrd.Interface) {
				// Try creating a house.
				h := &buildingv1.House{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
						Namespace: "default",
						Labels:    map[string]string{},
					},
					Spec: buildingv1.HouseSpec{
						Name:    "newHouse",
						Address: "whatever 42",
						Active:  &falseBool,
						Owners:  nil,
					},
				}
				_, err := crdcli.BuildingV1().Houses(h.Namespace).Create(context.TODO(), h, metav1.CreateOptions{})
				require.NoError(t, err)
				// nolint: errcheck
				defer crdcli.BuildingV1().Houses(h.Namespace).Delete(context.TODO(), h.Name, metav1.DeleteOptions{})

				// Check expectations.
				expHouseSpec := buildingv1.HouseSpec{
					Name:    "changed-name",
					Active:  &trueBool,
					Address: "",
					Owners: []buildingv1.User{
						{Name: "user1", Email: "user1@kubebwehook.slok.dev"},
					},
				}

				house, err := crdcli.BuildingV1().Houses(h.Namespace).Get(context.TODO(), h.Name, metav1.GetOptions{})
				if assert.NoError(t, err) {
					assert.Equal(t, expHouseSpec, house.Spec)
				}
			},
		},

		"(dynamic, CRD) Having a dynamic webhook, a mutation on a CRD creation should mutate the the CRD labels, rewrite the existing ones, and add the missing ones.": {
			webhookRegisterCfg: getMutatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesHouseCRD}, []string{version}),
			webhook: func() webhook.Webhook {
				// Our mutator logic.
				mut := mutating.MutatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*mutating.MutatorResult, error) {
					// Mutate.
					labels := obj.GetLabels()
					if labels == nil {
						labels = map[string]string{}
					}
					labels["city"] = "Madrid"
					labels["type"] = "Flat"
					labels["rooms"] = "3"
					obj.SetLabels(labels)

					return &mutating.MutatorResult{MutatedObject: obj}, nil
				})
				mwh, _ := mutating.NewWebhook(mutating.WebhookConfig{
					ID:      "house-mutator-label",
					Mutator: mut,
				})
				return mwh
			},
			execTest: func(t *testing.T, _ kubernetes.Interface, crdcli kubewebhookcrd.Interface) {
				// Try creating a house.
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
				expLabels := map[string]string{
					"city":      "Madrid",
					"bathrooms": "2",
					"rooms":     "3",
					"type":      "Flat",
				}
				house, err := crdcli.BuildingV1().Houses(h.Namespace).Get(context.TODO(), h.Name, metav1.GetOptions{})
				if assert.NoError(t, err) {
					assert.Equal(t, expLabels, house.Labels)
				}
			},
		},

		"(static, unstructured, CRD) Having a static webhook forcing unstructured, a mutation on a CRD creation should mutate the the CRD labels, rewrite the existing ones, and add the missing ones.": {
			webhookRegisterCfg: getMutatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesHouseCRD}, []string{version}),
			webhook: func() webhook.Webhook {
				// Our mutator logic.
				mut := mutating.MutatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*mutating.MutatorResult, error) {
					// Mutate.
					labels := obj.GetLabels()
					if labels == nil {
						labels = map[string]string{}
					}
					labels["city"] = "Madrid"
					labels["type"] = "Flat"
					labels["rooms"] = "3"
					obj.SetLabels(labels)

					return &mutating.MutatorResult{MutatedObject: obj}, nil
				})
				mwh, _ := mutating.NewWebhook(mutating.WebhookConfig{
					ID:      "house-mutator-label",
					Obj:     &unstructured.Unstructured{},
					Mutator: mut,
				})
				return mwh
			},
			execTest: func(t *testing.T, _ kubernetes.Interface, crdcli kubewebhookcrd.Interface) {
				// Try creating a house.
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
				expLabels := map[string]string{
					"city":      "Madrid",
					"bathrooms": "2",
					"rooms":     "3",
					"type":      "Flat",
				}
				house, err := crdcli.BuildingV1().Houses(h.Namespace).Get(context.TODO(), h.Name, metav1.GetOptions{})
				if assert.NoError(t, err) {
					assert.Equal(t, expLabels, house.Labels)
				}
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Register webhooks.
			_, err := cli.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), test.webhookRegisterCfg, metav1.CreateOptions{})
			require.NoError(t, err, "error registering webhooks kubernetes client")
			// nolint: errcheck
			defer cli.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.TODO(), test.webhookRegisterCfg.Name, metav1.DeleteOptions{})

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
			// nolint: errcheck
			defer srv.Shutdown(context.TODO())

			// Wait a bit to get ready with the webhook server goroutine.
			time.Sleep(2 * time.Second)

			// Execute test.
			test.execTest(t, cli, crdcli)
		})
	}
}

func testMutatingWebhookWarnings(t *testing.T) {
	cfg := helperconfig.GetTestEnvConfig(t)

	tests := map[string]struct {
		webhookRegisterCfg *arv1.MutatingWebhookConfiguration
		webhook            func() webhook.Webhook
		execTest           func(t *testing.T, cli kubernetes.Interface, crdcli kubewebhookcrd.Interface)
		expWarnings        string
	}{
		"Warning messages should be received by a mutating webhook.": {
			webhookRegisterCfg: getMutatingWebhookConfig(t, cfg, []arv1.RuleWithOperations{webhookRulesPod}, []string{"v1"}),
			webhook: func() webhook.Webhook {
				// Our mutator logic.
				mut := mutating.MutatorFunc(func(ctx context.Context, ar *model.AdmissionReview, obj metav1.Object) (*mutating.MutatorResult, error) {
					return &mutating.MutatorResult{Warnings: []string{
						"this is the first warning",
						"and this is the second warning",
					}}, nil
				})
				mwh, _ := mutating.NewWebhook(mutating.WebhookConfig{
					ID:      "pod-mutator-test2",
					Mutator: mut,
				})
				return mwh
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
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var gotWarnings bytes.Buffer

			cli, err := helpercli.GetK8sSTDClients(cfg.KubeConfigPath, &gotWarnings)
			require.NoError(t, err, "error getting kubernetes client")
			crdcli, err := helpercli.GetK8sCRDClients(cfg.KubeConfigPath, &gotWarnings)
			require.NoError(t, err, "error getting kubernetes CRD client")

			// Register webhooks.
			_, err = cli.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), test.webhookRegisterCfg, metav1.CreateOptions{})
			require.NoError(t, err, "error registering webhooks kubernetes client")
			// nolint: errcheck
			defer cli.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.TODO(), test.webhookRegisterCfg.Name, metav1.DeleteOptions{})

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

func TestMutatingWebhookV1Beta1(t *testing.T) {
	testMutatingWebhookCommon(t, "v1beta1")
}

func TestMutatingWebhookV1(t *testing.T) {
	testMutatingWebhookCommon(t, "v1")
	testMutatingWebhookWarnings(t)
}
