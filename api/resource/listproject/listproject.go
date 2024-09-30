package listproject

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/project"
)

var logger = log.New(os.Stderr, "RESOURCE LISTPROJECT: ", 0)

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
	Aliases              map[string]struct{}    `json:"aliases"`
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

	foundProjectDocs, err := database.GetMany[project.Project](transportCtx, routeCtx.DynamoClient, &database.GetManyInput{
		Table: aws.String(routeCtx.ProjectTable),
		Index: aws.String(project.GSI_OWNER_ID),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":owner_id": &dynamodbtypes.AttributeValueMemberS{Value: userToken.Id},
		},
		ConditionExpr: aws.String("owner_id = :owner_id"),
	})
	if err != nil {
		logger.Printf("failed load projects from database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed load projects from database")
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
			Aliases: project.Aliases,
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
