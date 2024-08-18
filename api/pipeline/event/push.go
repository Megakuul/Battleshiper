package event

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/go-playground/webhooks/v6/github"
	"github.com/google/uuid"
	"github.com/megakuul/battleshiper/api/pipeline/routecontext"
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
		bson.M{
			"owner_id":      userDoc.Id,
			"deleted":       false,
			"repository.id": event.Repository.ID,
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

	for _, proj := range foundProjectDocs {
		execIdentifier := uuid.New()
		eventResult := project.EventResult{
			ExecutionIdentifier: execIdentifier.String(),
			Timepoint:           time.Now().Unix(),
		}
		err := initiateProjectBuild(transportCtx, routeCtx, execIdentifier.String(), userDoc, &proj, subscriptionDoc)
		if err != nil {
			eventResult.Successful = false
			eventResult.EventOutput = err.Error()
		} else {
			eventResult.Successful = true
			eventResult.EventOutput = "project build was successfully initiated"
		}

		result, err := projectCollection.UpdateOne(transportCtx, bson.M{"name": proj.Name}, bson.M{
			"$set": bson.M{
				"last_event_result": eventResult,
			},
		})
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to update last_event_result on database")
		} else if result.MatchedCount < 1 {
			return http.StatusInternalServerError, fmt.Errorf("failed to update last_event_result: project not found")
		}
	}

	return http.StatusOK, nil
}

func initiateProjectBuild(
	transportCtx context.Context,
	routeCtx routecontext.Context,
	execIdentifier string,
	userDoc *user.User,
	projectDoc *project.Project,
	subscriptionDoc *subscription.Subscription) error {

	userCollection := routeCtx.Database.Collection(user.USER_COLLECTION)

	updatedUserDoc := &user.User{}
	err := userCollection.FindOneAndUpdate(transportCtx, bson.M{"id": userDoc.Id}, bson.M{
		"$set": bson.M{
			"limit_counter": bson.M{
				"pipeline_executions": bson.M{
					"$cond": bson.M{
						"if":   bson.M{"$lte": bson.A{"$limit_counter.expiration_time", time.Now().Unix()}},
						"then": 0,
						"else": bson.M{"$add": bson.A{"$limit_counter.pipeline_executions", 1}},
					},
				},
				"expiration_time": bson.M{
					"$cond": bson.M{
						"if":   bson.M{"$lte": bson.A{"$limit_counter.expiration_time", time.Now().Unix()}},
						"then": time.Now().Add(24 * time.Hour).Unix(),
						"else": "$limit_counter.expiration_time",
					},
				},
			},
		},
	}).Decode(updatedUserDoc)
	if err == mongo.ErrNoDocuments {
		return fmt.Errorf("failed to update user limit counter: user not found")
	} else if err != nil {
		return fmt.Errorf("failed to update user limit counter on database")
	}

	if updatedUserDoc.LimitCounter.PipelineExecutions > subscriptionDoc.DailyPipelineExecutions {
		return fmt.Errorf("subscription limit reached; no further pipeline executions can be performed")
	}

	buildRequest := &event.BuildRequest{
		ExecutionIdentifier:  execIdentifier,
		RepositoryURL:        projectDoc.Repository.URL,
		EventBusName:         routeCtx.EventBus,
		BuildCommand:         projectDoc.BuildCommand,
		BuildAssetBucketPath: projectDoc.BuildAssetBucketPath,
	}
	buildRequestRaw, err := json.Marshal(buildRequest)
	if err != nil {
		return fmt.Errorf("failed to serialize build request")
	}

	eventEntry := types.PutEventsRequestEntry{
		Source:       aws.String("ch.megakuul.battleshiper"),
		DetailType:   aws.String("battleshiper.build"),
		Detail:       aws.String(string(buildRequestRaw)),
		EventBusName: aws.String(routeCtx.EventBus),
	}
	res, err := routeCtx.EventClient.PutEvents(transportCtx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{eventEntry},
	})
	if err != nil || res.FailedEntryCount > 0 {
		return fmt.Errorf("failed to emit build event to the pipeline")
	}

	return nil
}
