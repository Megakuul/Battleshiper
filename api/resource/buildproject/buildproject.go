package buildproject

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
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

	// MIG: Possible with query item and primary key
	userDoc := &user.User{}
	err = userCollection.FindOne(transportCtx, bson.M{"id": userToken.Id}).Decode(&userDoc)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to load user record from database")
	}

	// MIG: Possible with query item and primary key + condition for owner_id (or later check in application code for owner)
	projectDoc := &project.Project{}
	err = projectCollection.FindOne(transportCtx, bson.D{
		{Key: "name", Value: buildProjectInput.ProjectName},
		{Key: "owner_id", Value: userDoc.Id},
	}).Decode(&projectDoc)
	if err == mongo.ErrNoDocuments {
		return nil, http.StatusNotFound, fmt.Errorf("project does not exist")
	} else if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update project on database")
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
		// MIG: Possible with query item and primary key
		result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
			"$set": bson.M{
				"last_event_result": eventResult,
				"status":            fmt.Errorf("EVENT FAILED: %v", err),
			},
		})
		if err != nil && result.MatchedCount < 1 {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to update project")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to initiate project build: %v", err)
	} else {
		eventResult.Successful = true
		eventResult.Timepoint = time.Now().Unix()
		// MIG: Possible with query item and primary key
		result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
			"$set": bson.M{
				"last_event_result": eventResult,
			},
		})
		if err != nil && result.MatchedCount < 1 {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to update project")
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
	cloudLogger.WriteLog("Emitting event to pipeline...")
	if err := cloudLogger.PushLogs(); err != nil {
		return err
	}

	err = emitBuildEvent(transportCtx, routeCtx, execId, userDoc, projectDoc)
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

func emitBuildEvent(transportCtx context.Context, routeCtx routecontext.Context, execId string, userDoc *user.User, projectDoc *project.Project) error {
	err := pipeline.CheckBuildSubscriptionLimit(transportCtx, routeCtx.Database, userDoc)
	if err != nil {
		return err
	}

	deployTicket, err := pipeline.CreateTicket(routeCtx.DeployTicketOptions, userDoc.Id, projectDoc.Name)
	if err != nil {
		return fmt.Errorf("failed to create pipeline ticket")
	}

	buildRequest := &event.BuildRequest{
		ExecutionIdentifier: execId,
		DeployTicket:        deployTicket,
		RepositoryURL:       projectDoc.Repository.URL,
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
