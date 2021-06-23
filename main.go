package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	apimetav1 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"time"

	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/fluxcd/pkg/apis/meta"
	log "github.com/sirupsen/logrus"
	helmaction "helm.sh/helm/v3/pkg/action"
	helmcli "helm.sh/helm/v3/pkg/cli"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
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
	var actionStr string
	var timeoutStr string
	var intervalStr string

	flag.StringVar(&spec, "spec", "", "Spec of the helmrelease object to apply")
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

	if spec == "" {
		log.Fatal("Spec is empty, unable to apply an empty spec on the cluster")
	}

	decodedSpec, err := base64.StdEncoding.DecodeString(spec)
	if err != nil {
		log.Fatalf("Failed to decode the string as a base64 string; got the string %v", spec)
	}
	log.Info("Successfully base64 decoded the spec")

	hr := &fluxhelmv2beta1.HelmRelease{}
	if err := yaml.Unmarshal(decodedSpec, hr); err != nil {
		log.Fatalf("Failed to decode the spec into yaml with the err %v", err)
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		log.Fatalf("Failed to initialize the client config with %v", err)
	}
	k8sScheme := scheme.Scheme
	if err := fluxhelmv2beta1.AddToScheme(k8sScheme); err != nil {
		log.Fatalf("Failed to add the flux helm scheme to the configuration scheme with %v", err)
	}
	clientSet, err := client.New(config, client.Options{Scheme: k8sScheme})
	if err != nil {
		log.Fatalf("Failed to create the clientset with the given config with %v", err)
	}

	if action == Install {
		installAction(ctx, cancel, clientSet, hr, interval)
	} else if action == Delete {
		deleteAction(ctx, cancel, clientSet, hr, interval)
	}
}

func installAction(ctx context.Context, cancel context.CancelFunc, clientSet client.Client, hr *fluxhelmv2beta1.HelmRelease, interval time.Duration) {
	defer cancel()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: hr.Namespace,
		},
	}

	temp := &corev1.Namespace{}
	if err := clientSet.Get(ctx, client.ObjectKeyFromObject(ns), temp); client.IgnoreNotFound(err) != nil {
		log.Fatalf("Failed to get instance of the namespace with %v", err)
	} else if err != nil {
		log.Infof("Did not find the namespace: %v, creating the namespace...", hr.Namespace)
		// Create the namespace if it doesn't exist
		if err := retry(ctx, func() error { return clientSet.Create(ctx, ns) }, interval); err != nil {
			log.Fatalf("retry got err: %v", err)
		}
	} else if temp.Status.Phase == corev1.NamespaceTerminating {
		log.Infof("Namespace: %v is in a terminating state, retrying to create until the namespace is deleted...", hr.Namespace)
		if err := retry(ctx, func() error { return clientSet.Create(ctx, ns) }, interval); err != nil {
			log.Fatalf("retry got err: %v", err)
		}
	}

	instance := &fluxhelmv2beta1.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hr.Name,
			Namespace: hr.Namespace,
		},
	}
	createOrUpdateFunc := func() error {
		result, err := controllerutil.CreateOrUpdate(ctx, clientSet, instance, func() error {
			instance.Annotations = hr.Annotations
			instance.Labels = hr.Labels
			instance.Spec = hr.Spec
			return nil
		})
		if err != nil {
			log.Errorf("failed to create or update the helm release with: %v", err)
			return err
		}
		log.Infof("succeeded creating or updating the helm release %s with the result %s", hr.Name, result)
		return nil
	}
	if err := retry(ctx, func() error { return createOrUpdateFunc() }, interval); err != nil {
		log.Fatalf("retry got err: %v", err)
	}
	if err := PollStatus(ctx, clientSet, types.NamespacedName{Name: hr.Name, Namespace: hr.Namespace}, interval, 5); err != nil {
		log.Fatalf("%v", err)
	}
}

func deleteAction(ctx context.Context, cancel context.CancelFunc, clientSet client.Client, hr *fluxhelmv2beta1.HelmRelease, interval time.Duration) {
	defer cancel()
	instance := &fluxhelmv2beta1.HelmRelease{}
	key := client.ObjectKey{
		Name:      hr.Name,
		Namespace: hr.Namespace,
	}
	if err := clientSet.Get(ctx, key, instance); client.IgnoreNotFound(err) != nil {
		log.Errorf("Failed to get instance of the helmrelease with %v", err)
		return
	} else if err != nil {
		// Unexpectedly, the object that we are trying to delete was not found
		// If this happens, we will log an error and return
		log.Errorf("Did not find the helm release: %v", hr.Name)
		return
	} else {
		log.Infof("Found the helm release: %v, deleting the release...", hr.Name)
		// Delete the HelmRelease
		if err := retry(ctx, func() error { return clientSet.Delete(ctx, hr) }, interval); err != nil {
			log.Errorf("retry got err: %v", err)
			return
		}
	}

	pollingFunc := func(done chan<- bool) {
		instance := &fluxhelmv2beta1.HelmRelease{}
		key := client.ObjectKey{
			Name:      hr.Name,
			Namespace: hr.Namespace,
		}
		if err := clientSet.Get(ctx, key, instance); client.IgnoreNotFound(err) != nil {
			log.Errorf("Failed to get instance of the helmrelease with %v", err)
			return
		} else if err != nil {
			log.Infof("Did not find the helm release: %v, the object is deleted", hr.Name)
			done <- true
		} else {
			log.Infof("Found the helm release: %v, the object is not deleted", hr.Name)
			return
		}
	}

	if err := poll(ctx, pollingFunc, time.Second*5); err != nil {
		log.Errorf("Failed to poll the deletion of the helm release with %v", err)
		// The helm release is still not deleted. Force cleanup as the final attempt
		forceCleanupHelmRelease(clientSet, hr, interval)
	}
}

// remove the finalizer from the helm release and execute force uninstall
func forceCleanupHelmRelease(clientSet client.Client, hr *fluxhelmv2beta1.HelmRelease, interval time.Duration) {
	// Create a new context
	ctx := context.Background()
	instance := &fluxhelmv2beta1.HelmRelease{}
	key := client.ObjectKey{
		Name:      hr.Name,
		Namespace: hr.Namespace,
	}
	if err := clientSet.Get(ctx, key, instance); client.IgnoreNotFound(err) != nil {
		log.Errorf("Failed to get instance of the helmrelease with %v", err)
	} else if err != nil {
		// The helm release was not found after the first failed cleanup attempt
		log.Errorf("Did not find the helm release: %v, the object is deleted", hr.Name)
	} else {
		log.Infof("Removing the helm release finalizer: %v...", fluxhelmv2beta1.HelmReleaseFinalizer)
		patch := client.MergeFrom(instance.DeepCopy())
		controllerutil.RemoveFinalizer(instance, fluxhelmv2beta1.HelmReleaseFinalizer)
		if err := clientSet.Patch(ctx, instance, patch); err != nil {
			log.Errorf("Failed to patch the helmrelease with %v", err)
			return
		}
		log.Infof("Uninstalling the release: %v...", hr.Name)
		settings := helmcli.New()
		actionConfig := new(helmaction.Configuration)
		if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), log.Infof); err != nil {
			log.Errorf("Failed to init the helm uninstall action with %v", err)
			return
		}
		client := helmaction.NewUninstall(actionConfig)
		if _, err := client.Run(hr.Name); err != nil {
			log.Errorf("Failed to uninstall the helm release with %v", err)
		}
	}
}

func PollStatus(ctx context.Context, clientSet client.Client, key types.NamespacedName, interval time.Duration, retrySeconds int) error {
	statusPoller := func(done chan<- bool) {
		instance := &fluxhelmv2beta1.HelmRelease{}
		if err := clientSet.Get(ctx, key, instance); err != nil {
			log.Infof("failed to get the instance %s from the api server", key.Name)
			return
		}
		if instance.Generation != instance.Status.ObservedGeneration {
			log.Infof("observed generation and current generation do not match for instance %s", key.Name)
			return
		}
		conditions := instance.GetStatusConditions()
		readyCondition := apimetav1.FindStatusCondition(*conditions, "Ready")
		if readyCondition == nil {
			log.Infof("cannot find a ready condition")
			return
		}

		var i int
		// Attempt to get the same status for 5 consecutive seconds
		for i = 0; i < retrySeconds; i++ {
			if readyCondition.Status != metav1.ConditionTrue || readyCondition.Reason != meta.ReconciliationSucceededReason {
				log.Infof("did not reach a ready condition")
				break
			}
			time.Sleep(time.Second)
		}
		if i == retrySeconds {
			done <- true
		}
	}

	// Poll the helm release condition consecutively until the timeout
	if err := poll(ctx, statusPoller, interval); err != nil {
		return fmt.Errorf("timed out waiting for condition")
	}
	return nil
}

// retry retries the passed retryable function, sleeping for the given backoffDuration
// If the context times out while executing the retryable function, an error is returned
func retry(ctx context.Context, retryable func() error, backoffDuration time.Duration) error {
	pollingFunc := func(done chan<- bool) {
		if err := retryable(); err != nil {
			log.Errorf("Failed to execute the retryable function with: %v", err)
		}
		log.Infof("successfully completed the retry function")
		done <- true
	}
	return poll(ctx, pollingFunc, backoffDuration)
}

// poll retries the poller function, sleeping for the given backoffDuration
// If the poller times out it will return an error
func poll(ctx context.Context, poller func(chan<- bool), backoffDuration time.Duration) error {
	done := make(chan bool)
	for {
		go poller(done)
		select {
		case <-ctx.Done():
			return fmt.Errorf("failed to complete polling within the timeout")
		case <-done:
			log.Infof("successfully completed the polling function")
			return nil
		default:
			time.Sleep(backoffDuration)
		}
	}
}
