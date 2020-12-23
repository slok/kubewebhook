package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	kubewebhookcrd "github.com/slok/kubewebhook/v2/test/integration/crd/client/clientset/versioned"
)

// GetK8sSTDClients returns a all k8s clients.
func GetK8sSTDClients(kubehome string, warningWriter io.Writer) (kubernetes.Interface, error) {
	// Try fallbacks.
	if kubehome == "" {
		if kubehome = os.Getenv("KUBECONFIG"); kubehome == "" {
			kubehome = filepath.Join(homedir.HomeDir(), ".kube", "config")
		}
	}

	// Load kubernetes local connection.
	config, err := clientcmd.BuildConfigFromFlags("", kubehome)
	if err != nil {
		return nil, fmt.Errorf("could not load configuration: %s", err)
	}

	if warningWriter != nil {
		config.WarningHandler = rest.NewWarningWriter(warningWriter, rest.WarningWriterOptions{Deduplicate: true})
	}

	// Get the client.
	k8sCli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return k8sCli, nil
}

// GetK8sCRDClients returns a all k8s clients.
func GetK8sCRDClients(kubehome string, warningWriter io.Writer) (kubewebhookcrd.Interface, error) {
	// Try fallbacks.
	if kubehome == "" {
		if kubehome = os.Getenv("KUBECONFIG"); kubehome == "" {
			kubehome = filepath.Join(homedir.HomeDir(), ".kube", "config")
		}
	}

	// Load kubernetes local connection.
	config, err := clientcmd.BuildConfigFromFlags("", kubehome)
	if err != nil {
		return nil, fmt.Errorf("could not load configuration: %s", err)
	}

	if warningWriter != nil {
		config.WarningHandler = rest.NewWarningWriter(warningWriter, rest.WarningWriterOptions{Deduplicate: true})
	}

	// Get the client.
	k8sCli, err := kubewebhookcrd.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not create crd client: %s", err)
	}

	return k8sCli, nil
}
