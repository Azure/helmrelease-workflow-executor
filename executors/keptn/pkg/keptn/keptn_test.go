package keptn

import (
	"testing"

	"github.com/keptn/go-utils/pkg/common/fileutils"
)

func TestNew(t *testing.T) {
	var git *Git
	// git := Git{
	// 	URL:   "http://20.81.14.4:3000",
	// 	User:  "gitea_admin",
	// 	Token: "",
	// }

	k, err := New("http://localhost:8083/api", "keptn", "keptn-api-token", git)
	if err != nil {
		t.Error(err)
	}

	t.Log(k.token)
}

func TestKeptn_CreateProject(t *testing.T) {
	// read the contents of a file
	bShipyard, err := fileutils.ReadFile("testwith/shipyard.yaml")
	if err != nil {
		t.Error(err)
	}

	var git *Git
	// git := Git{
	// 	URL:   "http://20.81.14.4:3000/gitea_admin/sample_app",
	// 	User:  "gitea_admin",
	// 	Token: "32e85151cead767a87f40c8ae89b25b54ac068b2",
	// }

	k, err := New("http://localhost:8083/api", "keptn", "keptn-api-token", git)
	if err != nil {
		t.Error(err)
	}

	err = k.CreateProject("test-project", bShipyard)
	if err != nil {
		t.Error(err)
	}

}

func TestKeptn_CreateService(t *testing.T) {
	var git *Git
	// git := Git{
	// 	URL:   "http://20.81.14.4:3000/gitea_admin/sample_app",
	// 	User:  "gitea_admin",
	// 	Token: "32e85151cead767a87f40c8ae89b25b54ac068b2",
	// }

	k, err := New("http://localhost:8083/api", "keptn", "keptn-api-token", git)
	if err != nil {
		t.Error(err)
	}

	err = k.CreateService("sample_app", "test-project")
	if err != nil {
		t.Error(err)
	}

}

func TestKeptn_AddResourceToAllStages(t *testing.T) {
	var git *Git
	// git := Git{
	// 	URL:   "http://20.81.14.4:3000/gitea_admin/sample_app",
	// 	User:  "gitea_admin",
	// 	Token: "32e85151cead767a87f40c8ae89b25b54ac068b2",
	// }

	k, err := New("http://localhost:8083/api", "keptn", "keptn-api-token", git)
	if err != nil {
		t.Error(err)
	}

	if err := k.AddResourceToAllStages("sample_app", "test-project", "sli.yaml", "testwith/sli.yaml"); err != nil {
		t.Error(err)
	}
}
