package keptn

import (
	"encoding/base64"
	"fmt"

	// "github.com/keptn/go-utils/pkg/api/models"

	keptnutils "github.com/keptn/go-utils/pkg/api/utils"
	"github.com/keptn/go-utils/pkg/common/fileutils"
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
	url             string
	git             Git
	token           *string
	apiHandler      *apiutils.APIHandler
	resourceHandler *apiutils.ResourceHandler
	projectHandler  *apiutils.ProjectHandler
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

	apiHandler := apiutils.NewAuthenticatedAPIHandler(url, t, "x-token", nil, "http")
	resourceHandler := apiutils.NewAuthenticatedResourceHandler(url, t, "x-token", nil, "http")
	projectHandler := apiutils.NewAuthenticatedProjectHandler(url, t, "x-token", nil, "http")

	return &Keptn{
		url:             url,
		token:           &t,
		apiHandler:      apiHandler,
		resourceHandler: resourceHandler,
		projectHandler:  projectHandler,
		git:             git,
	}, nil
}

func (k *Keptn) CreateProject(project string, shipyard []byte) error {
	encodedShipyardContent := base64.StdEncoding.EncodeToString(shipyard)
	if _, kErr := k.apiHandler.CreateProject(apimodels.CreateProject{
		GitRemoteURL: k.git.URL,
		GitToken:     k.git.Token,
		GitUser:      k.git.User,
		Name:         &project,
		Shipyard:     &encodedShipyardContent,
	}); kErr != nil {
		return fmt.Errorf("failed to create project with err: %v", kErr.GetMessage())
	}

	return nil
}

func (k *Keptn) CreateService(service, project string) error {
	if _, kErr := k.apiHandler.CreateService(project, apimodels.CreateService{
		ServiceName: &service,
	}); kErr != nil {
		return fmt.Errorf("failed to create service with err: %v", kErr.GetMessage())
	}

	return nil
}

func (k *Keptn) AddResourceToAllStages(service, project, resourceURI, localResourcePath string) error {
	stages, err := k.getProjectStages(project)
	if err != nil {
		return err
	}

	for _, stage := range stages {
		if err := k.AddResourceToStage(service, project, stage.StageName, resourceURI, localResourcePath); err != nil {
			return err
		}
	}

	return nil
}

func (k *Keptn) AddResourceToStage(service, project, stage, resourceURI, localResourcePath string) error {
	resourceContent, err := fileutils.ReadFileAsStr(localResourcePath)
	if err != nil {
		return err
	}
	resource := &apimodels.Resource{
		ResourceContent: resourceContent,
		ResourceURI:     &resourceURI,
	}
	if _, err := k.resourceHandler.CreateServiceResources(project, stage, service, []*apimodels.Resource{resource}); err != nil {
		return err
	}
	return nil
}

func (k *Keptn) getProjectStages(project string) ([]*apimodels.Stage, error) {
	p, kErr := k.projectHandler.GetProject(apimodels.Project{
		GitRemoteURI: k.git.URL,
		GitToken:     k.git.Token,
		GitUser:      k.git.User,
		ProjectName:  project,
	})
	if kErr != nil {
		return nil, fmt.Errorf("failed to get project with err: %v", kErr.GetMessage())
	}
	return p.Stages, nil
}
