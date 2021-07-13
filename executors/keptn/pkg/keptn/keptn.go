package keptn

import (
	"encoding/base64"
	"fmt"

	// "github.com/keptn/go-utils/pkg/api/models"

	keptnutils "github.com/keptn/go-utils/pkg/api/utils"
	keptnk8sutils "github.com/keptn/kubernetes-utils/pkg"

	apimodels "github.com/keptn/go-utils/pkg/api/models"
	apiutils "github.com/keptn/go-utils/pkg/api/utils"
)

type Git struct {
	URL   string
	Token string
	User  string
}

type Keptn struct {
	url   string
	git   Git
	token *string
}

func New(url, namespace, secretName string, git Git) (*Keptn, error) {
	// get token from secret
	t, err := keptnk8sutils.GetKeptnAPITokenFromSecret(false, namespace, secretName)
	if err != nil {
		return nil, err
	}

	// authenticate with the api server
	auth := keptnutils.NewAuthenticatedAuthHandler(url, t, "x-token", nil, "http")
	if _, kErr := auth.Authenticate(); kErr != nil {
		err = fmt.Errorf("failed to authenticate with err: %v", kErr)
		fmt.Printf("%#v", kErr.GetMessage())
		return nil, err
	}

	return &Keptn{
		url:   url,
		token: &t,
		git:   git,
	}, nil
}

func (k *Keptn) CreateProject(name string, shipyard []byte) error {
	apiHandler := apiutils.NewAuthenticatedAPIHandler(k.url, *k.token, "x-token", nil, "http")
	encodedShipyardContent := base64.StdEncoding.EncodeToString(shipyard)
	if _, kErr := apiHandler.CreateProject(apimodels.CreateProject{
		GitRemoteURL: k.git.URL,
		GitToken:     k.git.Token,
		GitUser:      k.git.User,
		Name:         &name,
		Shipyard:     &encodedShipyardContent,
	}); kErr != nil {
		return fmt.Errorf("failed to create project with err: %v", kErr.GetMessage())
	}

	return nil
}

func (k *Keptn) CreateService(name string) error {
	apiHandler := apiutils.NewAuthenticatedAPIHandler(k.url, *k.token, "x-token", nil, "http")
	if _, kErr := apiHandler.CreateService(name, apimodels.CreateService{
		ServiceName: &name,
	}); kErr != nil {
		return fmt.Errorf("failed to create service with err: %v", kErr.GetMessage())
	}

	return nil
}
