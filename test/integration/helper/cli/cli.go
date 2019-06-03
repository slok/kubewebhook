package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// GetK8sClients returns a all k8s clients.
func GetK8sClients(kubehome string) (kubernetes.Interface, error) {
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

	// Get the client.
	k8sCli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return k8sCli, nil
}
