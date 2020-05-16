package webhook_test

import (
	"testing"

	arv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	helperconfig "github.com/slok/kubewebhook/test/integration/helper/config"
)

func getMutatingWebhookConfig(t *testing.T, cfg helperconfig.TestEnvConfig, rules []arv1.RuleWithOperations) *arv1.MutatingWebhookConfiguration {
	whSideEffect := arv1.SideEffectClassNone
	var timeoutSecs int32 = 30
	return &arv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "integration-test-webhook",
		},
		Webhooks: []arv1.MutatingWebhook{
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

var (
	webhookRulesPod = arv1.RuleWithOperations{
		Operations: []arv1.OperationType{"CREATE"},
		Rule: arv1.Rule{
			APIGroups:   []string{""},
			APIVersions: []string{"v1"},
			Resources:   []string{"pods"},
		},
	}

	webhookRulesDeletePod = arv1.RuleWithOperations{
		Operations: []arv1.OperationType{"DELETE"},
		Rule: arv1.Rule{
			APIGroups:   []string{""},
			APIVersions: []string{"v1"},
			Resources:   []string{"pods"},
		},
	}

	webhookRulesHouseCRD = arv1.RuleWithOperations{
		Operations: []arv1.OperationType{"CREATE"},
		Rule: arv1.Rule{
			APIGroups:   []string{"building.kubewebhook.slok.dev"},
			APIVersions: []string{"v1"},
			Resources:   []string{"houses"},
		},
	}
)
