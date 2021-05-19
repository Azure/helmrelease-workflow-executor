package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"time"

	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/aggregator"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/yaml"
)

func main() {
	var spec string
	var timeoutStr string
	var intervalStr string

	flag.StringVar(&spec, "spec", "", "Spec of the helmrelease object to apply")
	flag.StringVar(&timeoutStr, "timeout", "5m", "Timeout for the execution of the argo workflow task")
	flag.StringVar(&intervalStr, "interval", "10s", "Retry interval for the all actions by the executor")
	flag.Parse()

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		log.Fatalf("Failed to parse timeout as a duration with %v", err)
	}
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		log.Fatalf("Failed to parse interval as a duration with %v", err)
	}
	log.Infof("Parsed the timeout: %v and the interval: %v", timeout.String(), interval.String())

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
		retry(ctx, func() error { return clientSet.Create(ctx, ns) }, interval)
	} else if temp.Status.Phase == corev1.NamespaceTerminating {
		log.Infof("Namespace: %v is in a terminating state, retrying to create until the namespace is deleted...", hr.Namespace)
		retry(ctx, func() error { return clientSet.Create(ctx, ns) }, interval)
	}

	instance := &fluxhelmv2beta1.HelmRelease{}
	key := client.ObjectKey{
		Name:      hr.Name,
		Namespace: hr.Namespace,
	}
	if err := clientSet.Get(ctx, key, instance); client.IgnoreNotFound(err) != nil {
		log.Fatalf("Failed to get instance of the helmrelease with %v", err)
	} else if err != nil {
		// This means that the object was not found
		log.Infof("Did not find the helm release: %v, creating the release...", hr.Name)
		// Create the HelmRelease if it doesn't exist
		retry(ctx, func() error { return clientSet.Create(ctx, hr) }, interval)
	} else {
		instance.Annotations = hr.Annotations
		instance.Labels = hr.Labels
		instance.Spec = hr.Spec

		log.Infof("Found the helm release: %v, updating the release...", hr.Name)
		// Update the HelmRelease
		retry(ctx, func() error { return clientSet.Update(ctx, instance) }, interval)
	}

	identifiers := object.ObjMetadata{
		Namespace: hr.Namespace,
		Name:      hr.Name,
		GroupKind: schema.GroupKind{
			Group: fluxhelmv2beta1.GroupVersion.Group,
			Kind:  fluxhelmv2beta1.HelmReleaseKind,
		},
	}

	// We give the poller two minutes before we time it out
	if err := PollStatus(ctx, cancel, clientSet, config, identifiers); err != nil {
		log.Fatalf("%v", err)
	}
}

func PollStatus(ctx context.Context, cancel context.CancelFunc, clientSet client.Client, config *rest.Config, identifiers ...object.ObjMetadata) error {
	defer cancel()

	restMapper, err := apiutil.NewDynamicRESTMapper(config)
	if err != nil {
		return err
	}
	poller := polling.NewStatusPoller(clientSet, restMapper)
	eventsChan := poller.Poll(ctx, identifiers, polling.Options{PollInterval: time.Second})

	coll := collector.NewResourceStatusCollector(identifiers)
	done := coll.ListenWithObserver(eventsChan, desiredStatusNotifierFunc(cancel))

	<-done

	if coll.Error != nil || ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("timed out waiting for condition")
	}

	return nil
}

func desiredStatusNotifierFunc(cancelFunc context.CancelFunc) collector.ObserverFunc {
	return func(rsc *collector.ResourceStatusCollector, _ event.Event) {
		var rss []*event.ResourceStatus
		for _, rs := range rsc.ResourceStatuses {
			rss = append(rss, rs)
		}
		aggStatus := aggregator.AggregateStatus(rss, status.CurrentStatus)
		log.Infof("Received an event from the helm release, aggregated status is: %v", aggStatus.String())
		if aggStatus == status.CurrentStatus {
			cancelFunc()
		}
	}
}

// retry retries the passed retryable function, sleeping for the given backoffDuration
// The function will return and exit with code 1 if the context times out
func retry(ctx context.Context, retryable func() error, backoffDuration time.Duration) {
	for {
		if err := retryable(); err != nil {
			log.Errorf("Failed to execute the retryable function with: %v", err)
			select {
			case <-ctx.Done():
				log.Fatalf("Failed to create a new namesapce within the timeout")
			default:
				time.Sleep(backoffDuration)
			}
		} else {
			// Get out of the retry loop if we have succeeded to execute the function
			break
		}
	}
}
