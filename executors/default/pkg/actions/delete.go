package actions

import (
	"context"
	"fmt"
	"github.com/Azure/orkestra-workflow-executor/executors/default/pkg/status"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	log "github.com/sirupsen/logrus"
	helmaction "helm.sh/helm/v3/pkg/action"
	helmcli "helm.sh/helm/v3/pkg/cli"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

func Delete(ctx context.Context, cancel context.CancelFunc, clientSet client.Client, hr *fluxhelmv2beta1.HelmRelease, interval time.Duration) error {
	defer cancel()
	instance := &fluxhelmv2beta1.HelmRelease{}
	key := client.ObjectKey{
		Name:      hr.Name,
		Namespace: hr.Namespace,
	}
	if err := clientSet.Get(ctx, key, instance); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get instance of the helm release with: %w", err)
	} else if err != nil {
		// Unexpectedly, the object that we are trying to delete was not found
		// If this happens, we will log an error and return
		log.Infof("did not find the helm release %s so no need to call delete", hr.Name)
		return nil
	} else {
		log.Infof("Found the helm release: %v, deleting the release...", hr.Name)
		// Delete the HelmRelease
		if err := status.Retry(ctx, func() error { return clientSet.Delete(ctx, hr) }, interval); err != nil {
			return fmt.Errorf("retry got error: %w", err)
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

	if err := status.Poll(ctx, pollingFunc, time.Second*5); err != nil {
		log.Errorf("Failed to poll the deletion of the helm release with %v", err)
		// The helm release is still not deleted. Force cleanup as the final attempt
		forceCleanupHelmRelease(clientSet, hr)
	}
	return nil
}

// remove the finalizer from the helm release and execute force uninstall
func forceCleanupHelmRelease(clientSet client.Client, hr *fluxhelmv2beta1.HelmRelease) {
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
