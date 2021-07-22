package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/orkestra-workflow-executor/executors/keptn/pkg/keptn"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
)

const (
	ShipyardFileName string = "shipyard.yaml"
)

func Install(ctx context.Context, cancel context.CancelFunc, hr *fluxhelmv2beta1.HelmRelease, interval time.Duration, data map[string]string, keptnConfig *keptn.KeptnConfig) error {
	keptnCli, err := keptn.New(keptnConfig.URL, keptnConfig.Namespace, keptnConfig.Token.SecretRef.Name, nil)
	if err != nil {
		return fmt.Errorf("failed to create the keptn client %w", err)
	}

	shipyard, ok := data[ShipyardFileName]
	if !ok {
		return fmt.Errorf("shipyard.yaml not found")
	}

	// if err := keptnCli.CreateProject(strings.ToLower(hr.Name+"-"+hr.Namespace), bShipyard); err != nil {
	if err := keptnCli.CreateProject(strings.ToLower("new-evaluation-project"), []byte(shipyard)); err != nil {
		return err
	}
	return nil
}
