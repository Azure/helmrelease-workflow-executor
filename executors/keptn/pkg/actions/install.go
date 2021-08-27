package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/orkestra-workflow-executor/executors/keptn/pkg/keptn"
	"github.com/Azure/orkestra-workflow-executor/executors/keptn/pkg/status"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
)

func Install(ctx context.Context, cancel context.CancelFunc, hr *fluxhelmv2beta1.HelmRelease, interval time.Duration, data map[string]string) error {
	keptnConfig := &keptn.KeptnConfig{}

	// Read the keptn-config.yaml file.
	// This file is required and cannot have empty fields
	if v, ok := data[keptn.KeptnConfigFileName]; !ok {
		return fmt.Errorf("failed to read plugin configuration from configmap")
	} else {
		if err := json.Unmarshal([]byte(v), keptnConfig); err != nil {
			return fmt.Errorf("failed to unmarshal the keptn configuration file into KeptnConfig object")
		}
	}

	if err := keptnConfig.Validate(); err != nil {
		return err
	}

	keptnCli, err := keptn.New(keptnConfig.URL, keptnConfig.Namespace, keptnConfig.Token.SecretRef.Name, nil)
	if err != nil {
		return fmt.Errorf("failed to create the keptn client %w", err)
	}

	shipyard, ok := data[keptn.ShipyardFileName]
	if !ok {
		return fmt.Errorf("shipyard.yaml not found")
	}

	appName := strings.ToLower(hr.Name + "-" + hr.Namespace)
	if err := keptnCli.CreateOrUpdateProject(appName, shipyard); err != nil {
		// if err := keptnCli.CreateProject(strings.ToLower("new-evaluation-project"), []byte(shipyard)); err != nil {
		return err
	}

	if err := keptnCli.CreateService(appName, appName); err != nil {
		return err
	}

	for k, v := range data {
		if err := keptnCli.AddResourceToAllStages(appName, appName, k, v); err != nil {
			return err
		}
	}

	if err := keptnCli.ConfigureMonitoring(appName, appName, "prometheus"); err != nil {
		return err
	}

	keptnCtx, err := keptnCli.TriggerEvaluation(appName, appName, keptnConfig.Timeframe)
	if err != nil {
		return err
	}

	// if err := status.Retry(ctx, func() error { return createOrUpdateFunc() }, interval); err != nil {
	// return fmt.Errorf("retry got error: %w", err)
	// }
	if err := pollStatus(ctx, keptnCli, keptnCtx, appName, appName, interval, 5); err != nil {
		return fmt.Errorf("failed to poll status with: %w", err)
	}
	return nil
}

func pollStatus(ctx context.Context, keptnCli *keptn.Keptn, keptnCtx, project, service string, interval time.Duration, retrySeconds int) error {
	statusPoller := func(done chan<- bool) {
		start := time.Now()
		defer func() {
			fmt.Printf("polling status finished execution in %v\n", time.Now().Sub(start))
		}()

		// fmt.Printf("Looking up event with keptnCtx : %s\n", keptnCtx)
		// lookup keptn evaluation triggered event status
		if err := keptnCli.GetEvents(service, project, keptnCtx); err != nil {
			if errors.Is(err, keptn.ErrEvaluationFailed) {
				fmt.Printf("evaluation failed with error : %v\n", err)
				log.Fatalf(err.Error())
			}
			return
		}

		done <- true
	}

	// Poll the helm release condition consecutively until the timeout
	if err := status.Poll(ctx, statusPoller, interval); err != nil {
		return fmt.Errorf("timed out waiting for condition")
	}
	return nil
}
