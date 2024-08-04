package event

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/webhooks/v6/github"
	"github.com/megakuul/battleshiper/api/pipeline/routecontext"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func handleRepoPush(transportCtx context.Context, routeCtx routecontext.Context, event github.PushPayload) (int, error) {
	installationId := event.Installation.ID

	userCollection := routeCtx.Database.Collection(user.USER_COLLECTION)

	userDoc := &user.User{}
	err := userCollection.FindOneAndUpdate(transportCtx, bson.M{"github_data.installation_id": installationId}, bson.M{
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
	}).Decode(&userDoc)
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

	if userDoc.LimitCounter.PipelineExecutions > subscriptionDoc.DailyPipelineExecutions {
		return http.StatusBadRequest, fmt.Errorf("subscription limit reached; no further pipeline executions can be performed")
	}

	// TODO Push event to next function

	return http.StatusOK, nil
}
