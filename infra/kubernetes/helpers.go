package kubernetes

import (
	"fmt"
	"os"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
)

const (
	// EnvKubernetesNamespace is the environment variable that contains the Kubernetes namespace
	EnvKubernetesNamespace = "K8S_POD_NAMESPACE"
	// EnvIsKubernetes is the environment variable that indicates if the application is running in a Kubernetes environment.
	EnvIsKubernetes = "IS_KUBERNETES"

	// EnvPodName is the environment variable that contains the name of the pod the application is running in.
	EnvPodName = "POD_NAME"
	// EnvNodeName is the environment variable that contains the name of the node the application is running on.
	EnvNodeName = "K8S_NODE_NAME"

	// EnvPodIP is the environment variable that contains the IP address of the pod the application is running in.
	envPodIP = "K8S_POD_IP"
)

// IsKubernetes returns true if the application is running in a Kubernetes environment.
func IsKubernetes() bool {
	// See helm/userclouds/templates/_helpers.tpl userclouds.envVars
	value, isK8s := os.LookupEnv(EnvIsKubernetes)
	return isK8s && string(value) == "true"
}

// GetPodName returns the name of the pod the application is running in.
func GetPodName() string {
	// See helm/userclouds/templates/_helpers.tpl userclouds.envVars
	return os.Getenv(EnvPodName)
}

// GetNodeName returns the name of the node the application is running on.
func GetNodeName() string {
	// See helm/userclouds/templates/_helpers.tpl userclouds.envVars
	return os.Getenv(EnvNodeName)
}

func getNamespace() string {
	// See helm/userclouds/templates/_helpers.tpl userclouds.envVars
	return os.Getenv(EnvKubernetesNamespace)
}

// GetPodIP returns the IP address of the pod the application is running in.
func GetPodIP() string {
	// See helm/userclouds/templates/_helpers.tpl userclouds.envVars
	return os.Getenv(envPodIP)
}

// GetHostForService returns the host name for to use for a given UC service in kubernetes.
func GetHostForService(svc service.Service) string {
	if svc.IsUndefined() {
		return ""
	}
	return GetHostFromServiceName(svc.ToServiceName())
}

// GetHostFromServiceName returns the host name for to use for a given service name in kubernetes.
func GetHostFromServiceName(serviceName string) string {
	ns := getNamespace()
	if ns == "" {
		return ""
	}
	// The naming of k8s services (and other objects) is different between on-prem and cloud (SasSS).
	// On-prem services are named userclouds-<service>.<namespace>.svc.cluster.local
	// Cloud services are named <service>.<namespace>.svc.cluster.local
	// This is because otherwise ArgoCD (stupidly) thinks that there are shared resources between ArgoCD apps
	// Even though they are in different namespaces
	if universe.Current().IsOnPrem() {
		return fmt.Sprintf("userclouds-%s.%s.svc.cluster.local", serviceName, ns)
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, ns)
}
