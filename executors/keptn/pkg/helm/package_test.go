package helm

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestHelm_Package(t *testing.T) {
	testLoc := os.TempDir()
	type args struct {
		resource v1.Object
	}
	tests := []struct {
		name    string
		h       *Helm
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "helmrelease",
			h: &Helm{
				loc: testLoc,
			},
			args: args{
				resource: defaultHelmRelease(t),
			},
			want:    filepath.Join(testLoc, defaultHelmRelease(t).Name+"-"+defaultHelmRelease(t).Spec.Chart.Spec.Version+".tgz"),
			wantErr: false,
		},
		{
			name: "not a helmrelease",
			h: &Helm{
				loc: testLoc,
			},
			args: args{
				resource: &appsv1.Deployment{
					ObjectMeta: v1.ObjectMeta{
						Name: "foo",
					},
					Spec:   appsv1.DeploymentSpec{},
					Status: appsv1.DeploymentStatus{},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "not a valid object",
			h: &Helm{
				loc: testLoc,
			},
			args: args{
				resource: nil,
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Helm{loc: tt.h.loc}
			got, err := h.Package(tt.args.resource)
			if (err != nil) != tt.wantErr {
				t.Errorf("helm.Package() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("helm.Package() = %v, want %v", got, tt.want)
			}
		})
	}
}

func defaultHelmRelease(t *testing.T) *fluxhelmv2beta1.HelmRelease {
	data, err := ioutil.ReadFile("testwith/helmrelease.yaml")
	if err != nil {
		t.Fatalf("failed to read default helmrelease : %v", err)
	}

	hr := &fluxhelmv2beta1.HelmRelease{}
	if err := yaml.Unmarshal(data, hr); err != nil {
		t.Fatalf("failed to unmarshal spec: %v", err)
	}

	return hr
}
