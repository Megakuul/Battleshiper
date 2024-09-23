package buildproject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/google/uuid"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/user"
)

type buildProjectInput struct {
	ProjectName string `json:"project_name"`
}

type buildProjectOutput struct {
	Message string `json:"message"`
}

// HandleBuildProject manually triggers a project build.
func HandleBuildProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleBuildProject(request, transportCtx, routeCtx)
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

func runHandleBuildProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*buildProjectOutput, int, error) {
	var buildProjectInput buildProjectInput
	err := json.Unmarshal([]byte(request.Body), &buildProjectInput)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to deserialize request: invalid body")
	}

	userTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("user_token")
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("no user_token provided")
	}

	userToken, err := auth.ParseJWT(routeCtx.JwtOptions, userTokenCookie.Value)
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("user_token is invalid: %v", err)
	}

	userDoc, err := database.GetSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: routeCtx.UserTable,
		Index: "",
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: userToken.Id},
		},
		ConditionExpr: "id = :id",
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("user not found")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load user record from database")
	}

	projectDoc, err := database.GetSingle[project.Project](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: routeCtx.ProjectTable,
		Index: "",
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":name": &dynamodbtypes.AttributeValueMemberS{Value: buildProjectInput.ProjectName},
		},
		ConditionExpr: "name = :name",
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("project not found")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load project from database")
	}

	if projectDoc.OwnerId != userDoc.Id {
		return nil, http.StatusForbidden, fmt.Errorf("you are not the owner of this project")
	}
	if !projectDoc.Initialized {
		return nil, http.StatusBadRequest, fmt.Errorf("project is not initialized")
	}
	if projectDoc.Deleted {
		return nil, http.StatusBadRequest, fmt.Errorf("project was already deleted")
	}

	execId := uuid.New().String()
	eventResult := project.EventResult{
		ExecutionIdentifier: execId,
	}
	if err = initiateProjectBuild(transportCtx, routeCtx, execId, userDoc, projectDoc); err != nil {
		eventResult.Successful = false
		eventResult.Timepoint = time.Now().Unix()

		eventResultAttributes, err := attributevalue.Marshal(&eventResult)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to serialize eventresult")
		}

		_, err = database.UpdateSingle[project.Project](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
			Table: routeCtx.ProjectTable,
			PrimaryKey: map[string]dynamodbtypes.AttributeValue{
				"name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.Name},
			},
			AttributeNames: map[string]string{
				"#last_event_result": "last_event_result",
				"#status":            "status",
			},
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":last_event_result": eventResultAttributes,
				":status":            &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("EVENT FAILED: %v", err)},
			},
			UpdateExpr: "SET #last_event_result = :last_event_result, #status = :status",
		})
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to update project: %v", err)
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to initiate project build: %v", err)
	} else {
		eventResult.Successful = true
		eventResult.Timepoint = time.Now().Unix()

		eventResultAttributes, err := attributevalue.Marshal(&eventResult)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to serialize eventresult")
		}

		_, err = database.UpdateSingle[project.Project](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
			Table: routeCtx.ProjectTable,
			PrimaryKey: map[string]dynamodbtypes.AttributeValue{
				"name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.Name},
			},
			AttributeNames: map[string]string{
				"#last_event_result": "last_event_result",
			},
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":last_event_result": eventResultAttributes,
			},
			UpdateExpr: "SET #last_event_result = :last_event_result",
		})
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to update project: %v", err)
		}
	}

	return &buildProjectOutput{
		Message: "project build initiated",
	}, http.StatusOK, nil
}

func initiateProjectBuild(transportCtx context.Context, routeCtx routecontext.Context, execId string, userDoc *user.User, projectDoc *project.Project) error {
	cloudLogger, err := pipeline.NewCloudLogger(transportCtx, routeCtx.CloudwatchClient, projectDoc.DedicatedInfrastructure.EventLogGroup, execId)
	if err != nil {
		return err
	}

	cloudLogger.WriteLog("START INIT %s", execId)
	cloudLogger.WriteLog("Event triggered by api request")
	if err := cloudLogger.PushLogs(); err != nil {
		return err
	}

	cloudLogger.WriteLog("Generating installation token...")
	installToken, _, err := routeCtx.GithubAppClient.Apps.CreateInstallationToken(transportCtx, userDoc.InstallationId, nil)
	if err != nil {
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return fmt.Errorf("failed to generate installation token: %v", err)
	}

	cloudLogger.WriteLog("Emitting event to pipeline...")
	err = emitBuildEvent(transportCtx, routeCtx, execId, installToken.GetToken(), userDoc, projectDoc)
	if err != nil {
		cloudLogger.WriteLog("failed to emit build event: %v", err)
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return fmt.Errorf("failed to emit build event: %v", err)
	} else {
		cloudLogger.WriteLog("project build was successfully initiated")
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
	}

	return nil
}

func emitBuildEvent(transportCtx context.Context, routeCtx routecontext.Context, execId, installToken string, userDoc *user.User, projectDoc *project.Project) error {
	err := pipeline.CheckBuildSubscriptionLimit(transportCtx, routeCtx.DynamoClient, &pipeline.CheckBuildSubscriptionLimitInput{
		UserTable:         routeCtx.UserTable,
		SubscriptionTable: routeCtx.SubscriptionTable,
		UserDoc:           *userDoc,
	})
	if err != nil {
		return err
	}

	deployTicket, err := pipeline.CreateTicket(routeCtx.DeployTicketOptions, userDoc.Id, projectDoc.Name)
	if err != nil {
		return fmt.Errorf("failed to create pipeline ticket")
	}

	// Usage of the installation token like this is documented here:
	// https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-as-a-github-app-installation
	tokenRepositoryUrl := fmt.Sprintf(
		"https://x-access-token:%s@%s",
		installToken,
		strings.TrimPrefix(projectDoc.Repository.URL, "https://"),
	)

	buildRequest := &event.BuildRequest{
		ExecutionIdentifier: execId,
		DeployTicket:        deployTicket,
		RepositoryURL:       tokenRepositoryUrl,
		RepositoryBranch:    projectDoc.Repository.Branch,
		BuildCommand:        projectDoc.BuildCommand,
		OutputDirectory:     projectDoc.OutputDirectory,
	}
	buildRequestRaw, err := json.Marshal(buildRequest)
	if err != nil {
		return fmt.Errorf("failed to serialize build request")
	}

	eventEntry := eventbridgetypes.PutEventsRequestEntry{
		Source:       aws.String(routeCtx.BuildEventOptions.Source),
		DetailType:   aws.String(routeCtx.BuildEventOptions.Action),
		Detail:       aws.String(string(buildRequestRaw)),
		EventBusName: aws.String(routeCtx.BuildEventOptions.EventBus),
	}
	res, err := routeCtx.EventClient.PutEvents(transportCtx, &eventbridge.PutEventsInput{
		Entries: []eventbridgetypes.PutEventsRequestEntry{eventEntry},
	})
	if err != nil || res.FailedEntryCount > 0 {
		return fmt.Errorf("failed to emit build event to the pipeline")
	}

	return nil
}
