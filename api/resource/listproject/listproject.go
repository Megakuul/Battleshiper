package listproject

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/project"
)

type eventResultOutput struct {
	ExecutionIdentifier string `json:"execution_identifier"`
	Timestamp           int64  `json:"timestamp"`
	Successful          bool   `json:"successful"`
}

type buildResultOutput struct {
	ExecutionIdentifier string `json:"execution_identifier"`
	Timestamp           int64  `json:"timestamp"`
	Successful          bool   `json:"successful"`
}

type deploymentResultOutput struct {
	ExecutionIdentifier string `json:"execution_identifier"`
	Timestamp           int64  `json:"timestamp"`
	Successful          bool   `json:"successful"`
}

type repositoryOutput struct {
	Id     int64  `json:"id"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

type projectOutput struct {
	Name                 string                 `json:"name"`
	Deleted              bool                   `json:"deleted"`
	Initialized          bool                   `json:"initialized"`
	Status               string                 `json:"status"`
	BuildImage           string                 `json:"build_image"`
	BuildCommand         string                 `json:"build_command"`
	OutputDirectory      string                 `json:"output_directory"`
	Repository           repositoryOutput       `json:"repository"`
	LastEventResult      eventResultOutput      `json:"last_event_result"`
	LastBuildResult      buildResultOutput      `json:"last_build_result"`
	LastDeploymentResult deploymentResultOutput `json:"last_deployment_result"`
}

type listProjectOutput struct {
	Message  string          `json:"message"`
	Projects []projectOutput `json:"projects"`
}

// HandleListProject performs a lookup for the projects that are owned by the user and returns them as json object.
func HandleListProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleListProject(request, transportCtx, routeCtx)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: code,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: err.Error(),
		}, nil
	}
	rawResponse, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "failed to serialize response",
		}, nil
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: code,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(rawResponse),
	}, nil
}

func runHandleListProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*listProjectOutput, int, error) {

	userTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("user_token")
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("no user_token provided")
	}

	userToken, err := auth.ParseJWT(routeCtx.JwtOptions, userTokenCookie.Value)
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("user_token is invalid: %v", err)
	}

	projectCollection := routeCtx.Database.Collection(project.PROJECT_COLLECTION)

	cursor, err := projectCollection.Find(transportCtx,
		bson.M{"owner_id": userToken.Id},
	)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch data from database")
	}

	foundProjectDocs := []project.Project{}
	err = cursor.All(transportCtx, &foundProjectDocs)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch and decode projects")
	}

	foundProjectOutput := []projectOutput{}
	for _, project := range foundProjectDocs {
		foundProjectOutput = append(foundProjectOutput, projectOutput{
			Name:            project.Name,
			Deleted:         project.Deleted,
			Initialized:     project.Initialized,
			Status:          project.Status,
			BuildImage:      project.BuildImage,
			BuildCommand:    project.BuildCommand,
			OutputDirectory: project.OutputDirectory,
			LastEventResult: eventResultOutput{
				ExecutionIdentifier: project.LastEventResult.ExecutionIdentifier,
				Timestamp:           project.LastEventResult.Timepoint,
				Successful:          project.LastEventResult.Successful,
			},
			LastBuildResult: buildResultOutput{
				ExecutionIdentifier: project.LastBuildResult.ExecutionIdentifier,
				Timestamp:           project.LastBuildResult.Timepoint,
				Successful:          project.LastBuildResult.Successful,
			},
			LastDeploymentResult: deploymentResultOutput{
				ExecutionIdentifier: project.LastDeploymentResult.ExecutionIdentifier,
				Timestamp:           project.LastDeploymentResult.Timepoint,
				Successful:          project.LastDeploymentResult.Successful,
			},
			Repository: repositoryOutput{
				Id:     project.Repository.Id,
				URL:    project.Repository.URL,
				Branch: project.Repository.Branch,
			},
		})
	}

	return &listProjectOutput{
		Message:  "projects fetched",
		Projects: foundProjectOutput,
	}, http.StatusOK, nil
}
