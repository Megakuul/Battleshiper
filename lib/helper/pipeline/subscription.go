package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// CheckBuildSubscriptionLimit atomically updates the users limit_counter and checks if more pipeline_builds can be performed.
// if an error occurs or the user has no pipeline_builds left an err is returned.
func CheckBuildSubscriptionLimit(transportCtx context.Context, database *mongo.Database, userDoc *user.User) error {
	subscriptionCollection := database.Collection(subscription.SUBSCRIPTION_COLLECTION)

	subscriptionDoc := &subscription.Subscription{}
	err := subscriptionCollection.FindOne(transportCtx, bson.M{"id": userDoc.SubscriptionId}).Decode(&subscriptionDoc)
	if err == mongo.ErrNoDocuments {
		return fmt.Errorf("user does not have a valid subscription associated")
	} else if err != nil {
		return fmt.Errorf("failed to fetch subscription from database")
	}

	userCollection := database.Collection(subscription.SUBSCRIPTION_COLLECTION)

	updatedUserDoc := &user.User{}
	err = userCollection.FindOneAndUpdate(transportCtx, bson.M{"id": userDoc.Id}, bson.M{
		"$set": bson.M{
			"limit_counter": bson.M{
				"pipeline_builds": bson.M{
					"$cond": bson.M{
						"if":   bson.M{"$lte": bson.A{"$limit_counter.pipeline_builds_exp", time.Now().Unix()}},
						"then": 0,
						"else": bson.M{"$add": bson.A{"$limit_counter.pipeline_builds", 1}},
					},
				},
				"pipeline_builds_exp": bson.M{
					"$cond": bson.M{
						"if":   bson.M{"$lte": bson.A{"$limit_counter.pipeline_builds_exp", time.Now().Unix()}},
						"then": time.Now().Add(24 * time.Hour).Unix(),
						"else": "$limit_counter.pipeline_builds_exp",
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

	if updatedUserDoc.LimitCounter.PipelineBuilds > subscriptionDoc.PipelineSpecs.DailyBuilds {
		return fmt.Errorf("subscription limit reached; no further pipeline builds can be performed")
	}

	return nil
}

// CheckDeploySubscriptionLimit atomically updates the users limit_counter and checks if more pipeline_deployments can be performed.
// if an error occurs or the user has no pipeline_deployments left an err is returned.
func CheckDeploySubscriptionLimit(transportCtx context.Context, database *mongo.Database, userDoc *user.User) error {
	subscriptionCollection := database.Collection(subscription.SUBSCRIPTION_COLLECTION)

	subscriptionDoc := &subscription.Subscription{}
	err := subscriptionCollection.FindOne(transportCtx, bson.M{"id": userDoc.SubscriptionId}).Decode(&subscriptionDoc)
	if err == mongo.ErrNoDocuments {
		return fmt.Errorf("user does not have a valid subscription associated")
	} else if err != nil {
		return fmt.Errorf("failed to fetch subscription from database")
	}

	userCollection := database.Collection(subscription.SUBSCRIPTION_COLLECTION)

	updatedUserDoc := &user.User{}
	err = userCollection.FindOneAndUpdate(transportCtx, bson.M{"id": userDoc.Id}, bson.M{
		"$set": bson.M{
			"limit_counter": bson.M{
				"pipeline_deployments": bson.M{
					"$cond": bson.M{
						"if":   bson.M{"$lte": bson.A{"$limit_counter.pipeline_deployments_exp", time.Now().Unix()}},
						"then": 0,
						"else": bson.M{"$add": bson.A{"$limit_counter.pipeline_deployments", 1}},
					},
				},
				"pipeline_deployments_exp": bson.M{
					"$cond": bson.M{
						"if":   bson.M{"$lte": bson.A{"$limit_counter.pipeline_deployments_exp", time.Now().Unix()}},
						"then": time.Now().Add(24 * time.Hour).Unix(),
						"else": "$limit_counter.pipeline_deployments_exp",
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

	if updatedUserDoc.LimitCounter.PipelineDeployments > subscriptionDoc.PipelineSpecs.DailyDeployments {
		return fmt.Errorf("subscription limit reached; no further pipeline deployments can be performed")
	}

	return nil
}
