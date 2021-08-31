package keptn

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	// "github.com/keptn/go-utils/pkg/api/models"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	apimodels "github.com/keptn/go-utils/pkg/api/models"
	apiutils "github.com/keptn/go-utils/pkg/api/utils"
	keptnlib "github.com/keptn/go-utils/pkg/lib"
	keptnk8sutils "github.com/keptn/kubernetes-utils/pkg"
)

var (
	ErrFailedDeleteProject = fmt.Errorf("failed to delete project")
	ErrEvaluationFailed    = fmt.Errorf("evaluation result shows failure")
)

type Keptn struct {
	url             string
	git             *Git
	token           *string
	apiHandler      *apiutils.APIHandler
	resourceHandler *apiutils.ResourceHandler
	projectHandler  *apiutils.ProjectHandler
	eventHandler    *apiutils.EventHandler
}

func New(url, namespace, secretName string, git *Git) (*Keptn, error) {
	if git == nil {
		log.Printf("No upstream git server provided. Using in-built git server")
	}

	// get token from secret
	t, err := keptnk8sutils.GetKeptnAPITokenFromSecret(true, namespace, secretName)
	if err != nil {
		return nil, err
	}

	apiHandler := apiutils.NewAuthenticatedAPIHandler(url, t, KeptnAuthTokenKey, nil, HTTPTransportProtocol)
	resourceHandler := apiutils.NewAuthenticatedResourceHandler(url, t, KeptnAuthTokenKey, nil, HTTPTransportProtocol)
	projectHandler := apiutils.NewAuthenticatedProjectHandler(url, t, KeptnAuthTokenKey, nil, HTTPTransportProtocol)
	eventHandler := apiutils.NewAuthenticatedEventHandler(url, t, KeptnAuthTokenKey, nil, HTTPTransportProtocol)

	return &Keptn{
		url:             url,
		token:           &t,
		apiHandler:      apiHandler,
		resourceHandler: resourceHandler,
		projectHandler:  projectHandler,
		eventHandler:    eventHandler,
		git:             git,
	}, nil
}

func (k *Keptn) CreateOrUpdateProject(project string, shipyard string) error {
	encodedShipyardContent := base64.StdEncoding.EncodeToString([]byte(shipyard))
	projectInfo := apimodels.CreateProject{
		Name:     &project,
		Shipyard: &encodedShipyardContent,
	}

	projectGetInfo := apimodels.Project{
		ProjectName: project,
	}

	if k.git != nil {
		projectInfo.GitRemoteURL = k.git.URL
		projectInfo.GitToken = k.git.Token
		projectInfo.GitUser = k.git.User

		projectGetInfo.GitRemoteURI = k.git.URL
		projectGetInfo.GitToken = k.git.Token
		projectGetInfo.GitUser = k.git.User
	}

	// FIXME: address comment
	// What happens if the Get call fails due to an actual error but the job exists?
	// Nitish Malhotra (08/31) - Error code lookup table is not provided by keptn.
	// There is no way to differentiate between errors unfortunately
	if _, kErr := k.projectHandler.GetProject(apimodels.Project{
		ProjectName:     project,
		ShipyardVersion: shipyard,
		Stages:          []*apimodels.Stage{},
	}); kErr == nil {
		fmt.Println("found the project - deleting it now")
		if _, kErr := k.apiHandler.DeleteProject(projectGetInfo); kErr != nil {
			return fmt.Errorf("failed to delete project with err: %v", kErr.GetMessage())
		}
	}

	if _, kErr := k.apiHandler.CreateProject(projectInfo); kErr != nil {
		return fmt.Errorf("failed to create project with err: %v", kErr.GetMessage())
	}

	return nil
}

func (k *Keptn) DeleteProject(project string) error {
	p := apimodels.Project{
		ProjectName: project,
	}

	if _, kErr := k.apiHandler.DeleteProject(p); kErr != nil {
		return fmt.Errorf("%v : %w", kErr.GetMessage(), ErrFailedDeleteProject)
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

func (k *Keptn) AddResourceToAllStages(service, project, resourceName, resourceContent string) error {
	stages, err := k.getProjectStages(project)
	if err != nil {
		return err
	}

	for _, stage := range stages {
		if err := k.AddResourceToStage(service, project, stage.StageName, resourceNameToURI(resourceName), resourceContent); err != nil {
			return err
		}
	}

	return nil
}

func (k *Keptn) AddResourceToStage(service, project, stage, resourceURI, resourceContent string) error {
	resource := &apimodels.Resource{
		ResourceContent: resourceContent,
		ResourceURI:     &resourceURI,
	}

	if _, err := k.resourceHandler.CreateServiceResources(project, stage, service, []*apimodels.Resource{resource}); err != nil {
		return err
	}
	return nil
}

func (k *Keptn) ConfigureMonitoring(project, service, monitoringType string) error {
	configureMonitoringEventData := &keptnlib.ConfigureMonitoringEventData{
		Type:    monitoringType,
		Project: project,
		Service: service,
	}

	source, _ := url.Parse("https://github.com/keptn/keptn/cli#configuremonitoring")

	sdkEvent := cloudevents.NewEvent()
	sdkEvent.SetID(uuid.New().String())
	sdkEvent.SetType(keptnlib.ConfigureMonitoringEventType)
	sdkEvent.SetSource(source.String())
	sdkEvent.SetDataContentType(cloudevents.ApplicationJSON)
	sdkEvent.SetData(cloudevents.ApplicationJSON, configureMonitoringEventData)

	eventByte, err := json.Marshal(sdkEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal cloud event. %s", err.Error())
	}

	apiEvent := apimodels.KeptnContextExtendedCE{}
	err = json.Unmarshal(eventByte, &apiEvent)
	if err != nil {
		return fmt.Errorf("failed to map cloud event to API event model. %s", err.Error())
	}

	_, kErr := k.apiHandler.SendEvent(apiEvent)
	if err != nil {
		return fmt.Errorf("sending configure-monitoring event was unsuccessful. %s", *kErr.Message)
	}

	return nil
}

func (k *Keptn) GetEvents(service, project, keptnCtx string) error {
	filter := &apiutils.EventFilter{
		Project:      project,
		Service:      service,
		KeptnContext: keptnCtx,
		EventType:    "sh.keptn.event.evaluation.finished",
	}

	eventsCtx, kErr := k.eventHandler.GetEvents(filter)
	if kErr != nil {
		return fmt.Errorf("failed to get events for keptn context %s. %s", keptnCtx, *kErr.Message)
	}

	if len(eventsCtx) != 1 {
		return fmt.Errorf("expected to see one event of type %s", filter.EventType)
	}

	if dataMap, ok := eventsCtx[0].Data.(map[string]interface{}); ok {
		result := dataMap["result"].(string)
		if result == "pass" {
			return nil
		}
		return ErrEvaluationFailed
	}
	return fmt.Errorf("event context data expected to be of type map[string]interface{}")
}

func (k *Keptn) TriggerEvaluation(service, project, timeframe string) (string, error) {
	currentTime := time.Now()

	evaluation := apimodels.Evaluation{
		Start:     currentTime.UTC().Format("2006-01-02T15:04:05"),
		Timeframe: timeframe,
	}

	stages, err := k.getProjectStages(project)
	if err != nil {
		return "", err
	}

	stage := stages[0].StageName

	eventCtx, kErr := k.apiHandler.TriggerEvaluation(project, stage, service, evaluation)
	if kErr != nil {
		return "", fmt.Errorf("failed to trigger evaluation with err: %v", kErr.GetMessage())
	}
	return *eventCtx.KeptnContext, nil
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
