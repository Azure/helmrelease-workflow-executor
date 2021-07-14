package helm

import (
	"errors"
	"os"
	"path/filepath"

	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type Helm struct {
	loc string
}

func packager() *Helm {
	h := &Helm{
		loc: os.TempDir(),
	}
	return h
}

func (h *Helm) Package(resource v1.Object) (string, error) {
	hr, ok := resource.(*fluxhelmv2beta1.HelmRelease)
	if !ok {
		return "", errors.New("k8s object expected to be of type HelmRelease")
	}

	ch, err := buildChart(hr)
	if err != nil {
		return "", err
	}

	return packageChart(ch, h.loc)
}

func (h *Helm) ChartDir() (string, error) {
	if h.loc == "" {
		return "", errors.New("chart directory not found")
	}
	return h.loc, nil
}

func buildChart(hr *fluxhelmv2beta1.HelmRelease) (*chart.Chart, error) {
	if hr == nil {
		return nil, errors.New("helmrelease object cannot be nil")
	}

	chartname := hr.Name
	data, err := yaml.Marshal(hr)
	if err != nil {
		return nil, err
	}

	// This is a helm chart that wraps the HelmRelease object for keptn to deploy through
	// its helm-service controller
	ch := &chart.Chart{
		Metadata: &chart.Metadata{
			Name:        chartname,
			Version:     hr.Spec.Chart.Spec.Version,
			Description: "Helm chart for HelmRelease object to be passed to keptn",
			APIVersion:  chart.APIVersionV2,
		},
		Templates: []*chart.File{{
			Name: filepath.Join("templates", hr.Name) + ".yaml",
			Data: data,
		}},
	}

	return ch, nil
}

func packageChart(ch *chart.Chart, dir string) (string, error) {
	if ch == nil {
		return "", errors.New("chart cannot be nil")
	}
	return chartutil.Save(ch, dir)
}
