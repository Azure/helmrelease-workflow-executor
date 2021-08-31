package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/orkestra-workflow-executor/executors/keptn/pkg/keptn"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Delete(ctx context.Context, cancel context.CancelFunc, clientSet client.Client, hr *fluxhelmv2beta1.HelmRelease, interval time.Duration, data map[string]string) error {
	keptnConfig := &keptn.Config{}

	// Read the keptn-config.yaml file.
	// This file is required and cannot have empty fields
	v, ok := data[keptn.KeptnConfigFileName]
	if !ok {
		return fmt.Errorf("failed to read plugin configuration from configmap")
	}
	if err := json.Unmarshal([]byte(v), keptnConfig); err != nil {
		return fmt.Errorf("failed to unmarshal the keptn configuration file into Config object")
	}

	if err := keptnConfig.Validate(); err != nil {
		return err
	}

	keptnCli, err := keptn.New(keptnConfig.URL, keptnConfig.Namespace, keptnConfig.Token.SecretRef.Name, nil)
	if err != nil {
		return fmt.Errorf("failed to create the keptn client %w", err)
	}
	appName := strings.ToLower(hr.Name + "-" + hr.Namespace)
	if err := keptnCli.DeleteProject(appName); err != nil {
		if errors.Is(err, keptn.ErrFailedDeleteProject) {
			return nil
		}
		return err
	}
	return nil
}
