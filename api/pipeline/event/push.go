package event

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
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
	installationId := event.Installation.ID

	userCollection := routeCtx.Database.Collection(user.USER_COLLECTION)

	userDoc := &user.User{}
	err := userCollection.FindOne(transportCtx, bson.M{"github_data.installation_id": installationId}).Decode(&userDoc)
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

	projectCursor, err := projectCollection.Find(transportCtx,
		bson.D{
			{Key: "repository.id", Value: event.Repository.ID},
			{Key: "owner_id", Value: userDoc.Id},
			{Key: "deleted", Value: false},
			{Key: "initialized", Value: true},
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
				},
			})
			if err != nil && result.MatchedCount < 1 {
				return http.StatusInternalServerError, fmt.Errorf("failed to update last_event_result")
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
				return http.StatusInternalServerError, fmt.Errorf("failed to update last_event_result")
			}
		}
	}

	return http.StatusOK, nil
}

func initiateProjectBuild(transportCtx context.Context, routeCtx routecontext.Context, execIdentifier string, userDoc *user.User, projectDoc *project.Project) error {
	logStreamIdentifier := fmt.Sprintf("%s/%s", time.Now().Format("2006/01/02"), execIdentifier)
	_, err := routeCtx.CloudwatchClient.CreateLogStream(transportCtx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(projectDoc.DedicatedInfrastructure.EventLogGroup),
		LogStreamName: aws.String(logStreamIdentifier),
	})
	if err != nil {
		return fmt.Errorf("failed to create logstream on %s", projectDoc.DedicatedInfrastructure.EventLogGroup)
	}

	logEvents := []cloudwatchtypes.InputLogEvent{{
		Message:   aws.String(fmt.Sprintf("START INIT %s", execIdentifier)),
		Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
	}}

	logEvents = append(logEvents, cloudwatchtypes.InputLogEvent{
		Message:   aws.String("Emitting event to pipeline..."),
		Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
	})

	err = emitBuildEvent(transportCtx, routeCtx, execIdentifier, userDoc, projectDoc)
	if err != nil {
		logEvents = append(logEvents, cloudwatchtypes.InputLogEvent{
			Message:   aws.String(fmt.Sprintf("failed to emit build event: %v", err)),
			Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
		})
		_, err = routeCtx.CloudwatchClient.PutLogEvents(transportCtx, &cloudwatchlogs.PutLogEventsInput{
			LogGroupName:  aws.String(projectDoc.DedicatedInfrastructure.EventLogGroup),
			LogStreamName: aws.String(logStreamIdentifier),
			LogEvents:     logEvents,
		})
		if err != nil {
			return fmt.Errorf("failed to send logevents to %s", projectDoc.DedicatedInfrastructure.EventLogGroup)
		}
		return fmt.Errorf("failed to emit build event: %v", err)
	} else {
		logEvents = append(logEvents, cloudwatchtypes.InputLogEvent{
			Message:   aws.String("project build was successfully initiated"),
			Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
		})
		_, err = routeCtx.CloudwatchClient.PutLogEvents(transportCtx, &cloudwatchlogs.PutLogEventsInput{
			LogGroupName:  aws.String(projectDoc.DedicatedInfrastructure.EventLogGroup),
			LogStreamName: aws.String(logStreamIdentifier),
			LogEvents:     logEvents,
		})
		if err != nil {
			return fmt.Errorf("failed to send logevents to %s", projectDoc.DedicatedInfrastructure.EventLogGroup)
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
