package keptn

import (
	"encoding/base64"
	"fmt"
	"log"

	// "github.com/keptn/go-utils/pkg/api/models"

	keptnutils "github.com/keptn/go-utils/pkg/api/utils"
	"github.com/keptn/go-utils/pkg/common/fileutils"
	keptnk8sutils "github.com/keptn/kubernetes-utils/pkg"

	apimodels "github.com/keptn/go-utils/pkg/api/models"
	apiutils "github.com/keptn/go-utils/pkg/api/utils"
	corev1 "k8s.io/api/core/v1"
)

type Git struct {
	URL   string
	Token string
	User  string
}

type Keptn struct {
	url             string
	git             *Git
	token           *string
	apiHandler      *apiutils.APIHandler
	resourceHandler *apiutils.ResourceHandler
	projectHandler  *apiutils.ProjectHandler
}

func New(url, namespace, secretName string, git *Git) (*Keptn, error) {
	if git == nil {
		log.Printf("No upstream git server provided. Using in-built git server")
	}

	// get token from secret
	t, err := keptnk8sutils.GetKeptnAPITokenFromSecret(false, namespace, secretName)
	if err != nil {
		return nil, err
	}

	// authenticate with the api server
	auth := keptnutils.NewAuthenticatedAuthHandler(url, t, "x-token", nil, "http")
	if _, kErr := auth.Authenticate(); kErr != nil {
		err = fmt.Errorf("failed to authenticate with err: %v", kErr)
		log.Printf("failed to authenticate with err : %v", kErr.GetMessage())
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
	createProject := apimodels.CreateProject{
		Name:     &project,
		Shipyard: &encodedShipyardContent,
	}

	if k.git != nil {
		createProject.GitRemoteURL = k.git.URL
		createProject.GitToken = k.git.Token
		createProject.GitUser = k.git.User
	}

	if _, kErr := k.apiHandler.CreateProject(createProject); kErr != nil {
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
	proj := apimodels.Project{
		ProjectName: project,
	}

	if k.git != nil {
		proj.GitRemoteURI = k.git.URL
		proj.GitToken = k.git.Token
		proj.GitUser = k.git.User
	}

	p, kErr := k.projectHandler.GetProject(proj)
	if kErr != nil {
		return nil, fmt.Errorf("failed to get project with err: %v", kErr.GetMessage())
	}
	return p.Stages, nil
}

type KeptnAPIToken struct {
	SecretRef *corev1.ObjectReference `json:"secretRef,omitempty"`
}

type KeptnConfig struct {
	URL       string        `json:"url,omitempty"`
	Namespace string        `json:"namespace,omitempty"`
	Token     KeptnAPIToken `json:"token,omitempty"`
}

func (k *KeptnConfig) Validate() error {
	if k.URL == "" {
		return fmt.Errorf("keptn API server (nginx) cannot be nil")
	}

	if k.Namespace == "" {
		return fmt.Errorf("keptn namespace must be specified")
	}

	if k.Token.SecretRef.Name == "" {
		return fmt.Errorf("keptn API token secret name must be specified")
	}

	return nil
}
