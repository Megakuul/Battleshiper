package event

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/go-playground/webhooks/v6/github"
	"github.com/google/uuid"
	"github.com/megakuul/battleshiper/api/pipeline/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func handleRepoPush(transportCtx context.Context, routeCtx routecontext.Context, event github.PushPayload) (int, error) {
	// events not triggered by branch pushes are not handled.
	if !strings.HasPrefix(event.Ref, "refs/heads/") {
		return http.StatusOK, nil
	}
	branch := strings.TrimPrefix(event.Ref, "refs/heads/")

	userCollection := routeCtx.Database.Collection(user.USER_COLLECTION)

	userDoc := &user.User{}
	err := userCollection.FindOne(transportCtx, bson.M{"github_data.installation_id": event.Installation.ID}).Decode(&userDoc)
	if err == mongo.ErrNoDocuments {
		return http.StatusNotFound, fmt.Errorf("user installation not found")
	} else if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to fetch user from database")
	}

	subscriptionCollection := routeCtx.Database.Collection(subscription.SUBSCRIPTION_COLLECTION)

	subscriptionDoc := &subscription.Subscription{}
	err = subscriptionCollection.FindOne(transportCtx, bson.M{"id": userDoc.SubscriptionId}).Decode(&subscriptionDoc)
	if err == mongo.ErrNoDocuments {
		return http.StatusNotFound, fmt.Errorf("user does not have a valid subscription associated")
	} else if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to fetch subscription from database")
	}

	projectCollection := routeCtx.Database.Collection(project.PROJECT_COLLECTION)

	// operation uses the defined compound index for {"repository.id", "owner_id", "repository.branch", "deleted"}
	projectCursor, err := projectCollection.Find(transportCtx,
		bson.D{
			{Key: "repository.id", Value: event.Repository.ID},
			{Key: "owner_id", Value: userDoc.Id},
			{Key: "repository.branch", Value: branch},
			{Key: "deleted", Value: false},
		},
	)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to fetch data from database")
	}

	foundProjectDocs := []project.Project{}
	err = projectCursor.All(transportCtx, &foundProjectDocs)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to fetch and decode projects")
	}

	for _, projectDoc := range foundProjectDocs {
		if !projectDoc.Initialized || projectDoc.Deleted {
			continue
		}

		execIdentifier := uuid.New().String()
		eventResult := project.EventResult{
			ExecutionIdentifier: execIdentifier,
		}
		if err = initiateProjectBuild(transportCtx, routeCtx, execIdentifier, userDoc, &projectDoc); err != nil {
			eventResult.Successful = false
			eventResult.Timepoint = time.Now().Unix()
			result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
				"$set": bson.M{
					"last_event_result": eventResult,
					"status":            fmt.Errorf("EVENT FAILED: %v", err),
				},
			})
			if err != nil && result.MatchedCount < 1 {
				return http.StatusInternalServerError, fmt.Errorf("failed to update project")
			}
			return http.StatusInternalServerError, fmt.Errorf("failed to initiate project build: %v", err)
		} else {
			eventResult.Successful = true
			eventResult.Timepoint = time.Now().Unix()
			result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
				"$set": bson.M{
					"last_event_result": eventResult,
				},
			})
			if err != nil && result.MatchedCount < 1 {
				return http.StatusInternalServerError, fmt.Errorf("failed to update project")
			}
		}
	}

	return http.StatusOK, nil
}

func initiateProjectBuild(transportCtx context.Context, routeCtx routecontext.Context, execIdentifier string, userDoc *user.User, projectDoc *project.Project) error {
	cloudLogger, err := pipeline.NewCloudLogger(transportCtx, routeCtx.CloudwatchClient, projectDoc.DedicatedInfrastructure.EventLogGroup, execIdentifier)
	if err != nil {
		return err
	}

	cloudLogger.WriteLog("START INIT %s", execIdentifier)
	cloudLogger.WriteLog("Emitting event to pipeline...")
	if err := cloudLogger.PushLogs(); err != nil {
		return err
	}

	err = emitBuildEvent(transportCtx, routeCtx, execIdentifier, userDoc, projectDoc)
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

func emitBuildEvent(transportCtx context.Context, routeCtx routecontext.Context, execIdentifier string, userDoc *user.User, projectDoc *project.Project) error {
	err := pipeline.CheckBuildSubscriptionLimit(transportCtx, routeCtx.Database, userDoc)
	if err != nil {
		return err
	}

	deployTicket, err := pipeline.CreateTicket(routeCtx.DeployTicketOptions, userDoc.Id, projectDoc.Name)
	if err != nil {
		return fmt.Errorf("failed to create pipeline ticket")
	}

	buildRequest := &event.BuildRequest{
		ExecutionIdentifier: execIdentifier,
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
