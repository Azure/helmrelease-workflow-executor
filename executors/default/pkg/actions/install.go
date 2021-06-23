package actions

import (
	"context"
	"fmt"
	"github.com/Azure/orkestra-workflow-executor/executors/default/pkg/status"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/fluxcd/pkg/apis/meta"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apimetav1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

func Install(ctx context.Context, cancel context.CancelFunc, clientSet client.Client, hr *fluxhelmv2beta1.HelmRelease, interval time.Duration) error {
	defer cancel()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: hr.Namespace,
		},
	}

	temp := &corev1.Namespace{}
	if err := clientSet.Get(ctx, client.ObjectKeyFromObject(ns), temp); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get instance of the namespace: %w", err)
	} else if err != nil {
		log.Infof("Did not find the namespace: %v, creating the namespace...", hr.Namespace)
		// Create the namespace if it doesn't exist
		if err := status.Retry(ctx, func() error { return clientSet.Create(ctx, ns) }, interval); err != nil {
			return fmt.Errorf("retry got error %w", err)
		}
	} else if temp.Status.Phase == corev1.NamespaceTerminating {
		log.Infof("Namespace: %v is in a terminating state, retrying to create until the namespace is deleted...", hr.Namespace)
		if err := status.Retry(ctx, func() error { return clientSet.Create(ctx, ns) }, interval); err != nil {
			return fmt.Errorf("retry got error: %w", err)
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
	if err := status.Retry(ctx, func() error { return createOrUpdateFunc() }, interval); err != nil {
		return fmt.Errorf("retry got error: %w", err)
	}
	if err := pollStatus(ctx, clientSet, types.NamespacedName{Name: hr.Name, Namespace: hr.Namespace}, interval, 5); err != nil {
		return fmt.Errorf("failed to poll status with: %w", err)
	}
	return nil
}

func pollStatus(ctx context.Context, clientSet client.Client, key types.NamespacedName, interval time.Duration, retrySeconds int) error {
	statusPoller := func(done chan<- bool) {
		start := time.Now()
		defer func() {
			log.Infof("polling status finished execution in %v", time.Now().Sub(start))
		}()

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
	if err := status.Poll(ctx, statusPoller, interval); err != nil {
		return fmt.Errorf("timed out waiting for condition")
	}
	return nil
}
