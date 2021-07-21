package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Azure/orkestra-workflow-executor/executors/keptn/pkg/actions"
	"github.com/Azure/orkestra-workflow-executor/executors/keptn/pkg/keptn"

	log "github.com/sirupsen/logrus"
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
	var cmName, cmNamespace string
	var keptnURL, keptnNS, keptnSecretName string
	var addResourcePath string
	var actionStr string
	var timeoutStr string
	var intervalStr string

	flag.StringVar(&cmNamespace, "cm-namespace", "", "namespace of the configmap containing the shipyard.yaml and other resources")
	flag.StringVar(&cmName, "cm-name", "", "name of the configmap containing the shipyard.yaml and other resources")
	flag.StringVar(&addResourcePath, "add-resource-path", ".", "the target path to stage the resources from the configmap")
	flag.StringVar(&keptnURL, "keptn-url", "", "keptn API service URL")
	flag.StringVar(&keptnNS, "keptn-namespace", "", "keptn API service namespace")
	flag.StringVar(&keptnSecretName, "keptn-secret", "", "keptn API service secret that contains the X_TOKEN")
	flag.StringVar(&actionStr, "action", "", "Action to perform on the helmrelease object. Must be either install or delete")
	flag.StringVar(&timeoutStr, "timeout", "5m", "Timeout for the execution of the argo workflow task")
	flag.StringVar(&intervalStr, "interval", "10s", "Retry interval for the all actions by the executor")
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
	log.Infof("Parsed the action: %v, the timeout: %v and the interval: %v", string(action), timeout.String(), interval.String())

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		log.Fatalf("Failed to initialize the client config with %v", err)
	}
	k8sScheme := scheme.Scheme

	clientSet, err := client.New(config, client.Options{Scheme: k8sScheme})
	if err != nil {
		log.Fatalf("Failed to create the clientset with the given config with %v", err)
	}

	keptnCli, err := keptn.New(keptnURL, keptnNS, keptnSecretName, nil)
	if err != nil {
		log.Fatalf("Failed to create the keptn client %v", err)
	}

	if action == Install {
		if err := os.MkdirAll(addResourcePath, 0755); err != nil {
			log.Fatalf("Failed to create the directory %v with %v", addResourcePath, err)
		}

		cm := types.NamespacedName{
			Name:      cmName,
			Namespace: cmNamespace,
		}
		if err := actions.Install(ctx, cancel, addResourcePath, clientSet, keptnCli, cm, interval); err != nil {
			log.Fatalf("failed to trigger keptn evaluation: %v", err)
		}
	} else if action == Delete {
		if err := actions.Delete(ctx, cancel, clientSet, interval); err != nil {
			log.Fatalf("failed to cleanup keptn application resources: %v", err)
		}
	}
}
