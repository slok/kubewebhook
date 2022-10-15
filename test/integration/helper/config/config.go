package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/util/homedir"
)

const (
	envVarWebhookURL  = "TEST_WEBHOOK_URL"
	envVarListenPort  = "TEST_LISTEN_PORT"
	envVarCertPath    = "TEST_CERT_PATH"
	envVarCertKeyPath = "TEST_CERT_KEY_PATH"
	envKubeConfig     = "KUBECONFIG"
)

// TestEnvConfig has the integration tests environment configuration.
type TestEnvConfig struct {
	WebhookURL         string
	ListenAddress      string
	KubeConfigPath     string
	WebhookCertPath    string
	WebhookCertKeyPath string
	WebhookCert        string
}

func (c *TestEnvConfig) defaults() error {
	if c.WebhookCertPath == "" {
		c.WebhookCertPath = "../certs/cert.pem"
	}

	if c.WebhookCertKeyPath == "" {
		c.WebhookCertKeyPath = "../certs/key.pem"
	}

	// Load certificate data.
	if c.WebhookCert == "" {
		cert, err := os.ReadFile(c.WebhookCertPath)
		if err != nil {
			return fmt.Errorf("error loading cert: %s", err)
		}
		c.WebhookCert = string(cert)
	}

	if c.ListenAddress == "" || c.ListenAddress == ":" {
		c.ListenAddress = ":8080"
	}

	if c.KubeConfigPath == "" {
		c.KubeConfigPath = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	// To create a local testing development env you could do:
	// - `kind create cluster`
	// - `ssh -R 0:localhost:8080 tunnel.us.ngrok.com tcp 22`
	// Use the `https://0.tcp.ngrok.io:17661` url style as `TEST_WEBHOOK_URL` env var.
	if c.WebhookURL == "" {
		return fmt.Errorf("webhook url is required")
	}

	return nil
}

// GetTestEnvConfig returns the configuration that should have the environment
// so the integration tests can be run.
func GetTestEnvConfig(t *testing.T) TestEnvConfig {
	cfg := TestEnvConfig{
		WebhookURL:         os.Getenv(envVarWebhookURL),
		ListenAddress:      fmt.Sprintf(":%s", os.Getenv(envVarListenPort)),
		KubeConfigPath:     os.Getenv(envKubeConfig),
		WebhookCertPath:    os.Getenv(envVarCertPath),
		WebhookCertKeyPath: os.Getenv(envVarCertKeyPath),
	}

	err := cfg.defaults()
	if err != nil {
		t.Fatalf("could not load integration tests configuration: %s", err)
	}

	return cfg
}
