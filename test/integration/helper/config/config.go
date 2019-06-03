package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

const (
	envVarWebhookURL = "TEST_WEBHOOK_URL"
	envVarListenPort = "TEST_LISTEN_PORT"
	envKubeConfig    = "KUBECONFIG"
	certPath         = "../certs/cert.pem"
	certKeyPath      = "../certs/key.pem"
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

// GetTestDevelopmentEnvConfig gives a test env configuration that is
// helpful to develop the integration tests, instead of executing
// all the time through the main and creating and destroying the
// kubernets clusters, SSH tunnels...
// To create the test development environment do:
// Run k3s:
//	sudo k3s server
// Run serveo on a random serveo subdomain (this example `slok-kubewebhook)
// and :8080 address:
// 	ssh -R slok-kubewebhook:1987:localhost:8080 serveo.net
func GetTestDevelopmentEnvConfig(t *testing.T) TestEnvConfig {
	os.Setenv(envVarWebhookURL, "https://slok-kubewebhook.serveo.net:1987")
	os.Setenv(envVarListenPort, "8080")
	os.Setenv(envKubeConfig, "/etc/rancher/k3s/k3s.yaml")

	return GetTestEnvConfig(t)
}

// GetTestEnvConfig returns the configuration that should have the environment
// so the integration tests can be run.
func GetTestEnvConfig(t *testing.T) TestEnvConfig {
	// Load files.
	cert, err := ioutil.ReadFile(certPath)
	if err != nil {
		t.Fatalf("error loading cert: %s", err)
	}

	return TestEnvConfig{
		WebhookURL:         os.Getenv(envVarWebhookURL),
		ListenAddress:      fmt.Sprintf(":%s", os.Getenv(envVarListenPort)),
		KubeConfigPath:     os.Getenv(envKubeConfig),
		WebhookCertPath:    certPath,
		WebhookCertKeyPath: certKeyPath,
		WebhookCert:        string(cert),
	}
}
