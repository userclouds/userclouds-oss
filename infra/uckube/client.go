package uckube

import (
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient configures and returns a kubernetes client interface
func NewClient() (kubernetes.Interface, error) {
	var kubeconfig *rest.Config

	if kcfg := os.Getenv("KUBECONFIG"); kcfg != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kcfg)
		if err != nil {
			return nil, err
		}
		kubeconfig = config
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		kubeconfig = config
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}
