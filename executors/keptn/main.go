package main

// https://socialsign.in/spot?nbiIP=100.124.9.55&loc=4176656e74696e65&client_mac=38:F9:D3:B5:DD:FB&domain_name=San+Diego&reason=Un-Auth-Captive&lid=network%3Dsd02&wlanName=WeWorkGuest&dn=scg.ruckuswireless.com&ssid=WeWorkGuest&mac=44:1e:98:39:aa:b0&url=http%3A%2F%2F10.239.22.1%2F&proxy=0&vlan=22&wlan=19&sip=scg.ruckuswireless.com&zoneName=tS-IA_qoXMRjSxWeD6KI8A_1627936773232&apip=10.239.12.175&sshTunnelStatus=1&uip=10.239.22.176&StartURL=https%3A%2F%2Fsocialsign.in%2Fspot%2Fsuccess
import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"time"

	"github.com/Azure/orkestra-workflow-executor/executors/keptn/pkg/actions"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"sigs.k8s.io/yaml"

	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// The set of executor actions which can be performed on a helmrelease object
	Install ExecutorAction = "install"
	Delete  ExecutorAction = "delete"
)

// ExecutorAction defines the set of executor actions which can be performed on a helmrelease object
type ExecutorAction string

func ParseExecutorAction(s string) (ExecutorAction, error) {
	a := ExecutorAction(s)
	switch a {
	case Install, Delete:
		return a, nil
	}
	return "", fmt.Errorf("invalid executor action: %v", s)
}

func main() {
	var spec string
	var configMapName, configMapNamespace string
	var actionStr string
	var timeoutStr string
	var intervalStr string

	// Default executor params
	flag.StringVar(&spec, "spec", "", "Spec of the helmrelease object to apply")
	flag.StringVar(&actionStr, "action", "", "Action to perform on the helmrelease object. Must be either install or delete")
	flag.StringVar(&timeoutStr, "timeout", "5m", "Timeout for the execution of the argo workflow task")
	flag.StringVar(&intervalStr, "interval", "10s", "Retry interval for the all actions by the executor")

	// Executor specific params
	flag.StringVar(&configMapName, "configmap-name", "", "name of the configmap containing the shipyard.yaml, plugin configuration and other resources")
	flag.StringVar(&configMapNamespace, "configmap-namespace", "", "namespace of the configmap containing the shipyard.yaml, plugin configuration and other resources")

	flag.Parse()

	action, err := ParseExecutorAction(actionStr)
	if err != nil {
		log.Fatalf("Failed to parse action as an executor action with %v", err)
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		log.Fatalf("Failed to parse timeout as a duration with %v", err)
	}
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		log.Fatalf("Failed to parse interval as a duration with %v", err)
	}
	log.Printf("Parsed the action: %v, the timeout: %v and the interval: %v", string(action), timeout.String(), interval.String())

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		log.Fatalf("Failed to initialize the client config with %v", err)
	}

	decodedSpec, err := base64.StdEncoding.DecodeString(spec)
	if err != nil {
		log.Fatalf("Failed to decode the string as a base64 string; got the string %v", spec)
	}
	log.Printf("Successfully base64 decoded the spec")

	hr := &fluxhelmv2beta1.HelmRelease{}
	if err := yaml.Unmarshal(decodedSpec, hr); err != nil {
		log.Fatalf("Failed to decode the spec into yaml with the err %v", err)
	}

	k8sScheme := scheme.Scheme
	clientSet, err := client.New(config, client.Options{Scheme: k8sScheme})
	if err != nil {
		log.Fatalf("Failed to create the clientset with the given config with %v", err)
	}

	configmapObj := &corev1.ConfigMap{}
	if err := clientSet.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: configMapNamespace}, configmapObj); err != nil {
		log.Fatalf("failed to get ConfigMap : %v", err)
	}

	if configmapObj.Data == nil {
		log.Fatalf("configmap data field cannot be nil")
	}

	if len(configmapObj.Data) == 0 {
		log.Fatalf("configmap data field cannot be empty")
	}

	if action == Install {
		if err := actions.Install(ctx, cancel, hr, interval, configmapObj.Data); err != nil {
			log.Fatalf("failed to trigger keptn evaluation: %v", err)
		}
	} else if action == Delete {
		if err := actions.Delete(ctx, cancel, clientSet, hr, interval, configmapObj.Data); err != nil {
			log.Fatalf("failed to cleanup keptn application resources: %v", err)
		}
	}
}
