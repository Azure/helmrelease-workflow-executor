package packager

import (
	"errors"
	"os"

	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	yamlv2 "gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type helm struct {
	loc string
}

func Helm() *helm {
	h := &helm{
		loc: os.TempDir(),
	}
	return h
}

func (h *helm) Package(resource v1.Object) (string, error) {
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

func (h *helm) ChartDir() (string, error) {
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
	data, err := yamlv2.Marshal(hr)
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
			Name: hr.Name,
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
