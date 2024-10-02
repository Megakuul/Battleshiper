package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
)

type CheckBuildSubscriptionLimitInput struct {
	UserTable         string
	SubscriptionTable string
	UserDoc           user.User
}

// CheckBuildSubscriptionLimit updates the users limit_counter and checks if more pipeline_builds can be performed.
// If the limit_counter values have expired, the pipeline_builds are reset and the expiration time is set to the next day.
func CheckBuildSubscriptionLimit(transportCtx context.Context, dynamoClient *dynamodb.Client, input *CheckBuildSubscriptionLimitInput) error {
	subscriptionDoc, err := database.GetSingle[subscription.Subscription](transportCtx, dynamoClient, &database.GetSingleInput{
		Table: aws.String(input.SubscriptionTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: input.UserDoc.SubscriptionId},
		},
		ConditionExpr: aws.String("id = :id"),
	})
	if err != nil {
		return fmt.Errorf("failed to fetch subscription: %v", err)
	}

	// Update pipeline_builds if the current counter has expired.
	_, err = database.UpdateSingle[user.User](transportCtx, dynamoClient, &database.UpdateSingleInput{
		Table: aws.String(input.UserTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: input.UserDoc.Id},
		},
		Upsert:    false,
		ReturnOld: false,
		AttributeNames: map[string]string{
			"#limit_counter":       "limit_counter",
			"#pipeline_builds":     "pipeline_builds",
			"#pipeline_builds_exp": "pipeline_builds_exp",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":pipeline_builds": &dynamodbtypes.AttributeValueMemberN{Value: "0"},
			":current_timestamp": &dynamodbtypes.AttributeValueMemberN{Value: strconv.Itoa(
				int(time.Now().Unix()),
			)},
			":next_timestamp": &dynamodbtypes.AttributeValueMemberN{Value: strconv.Itoa(
				int(time.Now().Add(24 * time.Hour).Unix()),
			)},
		},
		ConditionExpr: aws.String("#limit_counter.#pipeline_builds_exp < :current_timestamp"),
		UpdateExpr:    aws.String("SET #limit_counter.#pipeline_builds = :pipeline_builds, #limit_counter.#pipeline_builds_exp = :next_timestamp"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); !ok {
			return fmt.Errorf("failed to reset limit_counter: %v", err)
		}
	}

	updatedUserDoc, err := database.UpdateSingle[user.User](transportCtx, dynamoClient, &database.UpdateSingleInput{
		Table: aws.String(input.UserTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: input.UserDoc.Id},
		},
		Upsert:    false,
		ReturnOld: false,
		AttributeNames: map[string]string{
			"#limit_counter":   "limit_counter",
			"#pipeline_builds": "pipeline_builds",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":increment": &dynamodbtypes.AttributeValueMemberN{Value: "1"},
		},
		UpdateExpr: aws.String("ADD #limit_counter.#pipeline_builds :increment"),
	})
	if err != nil {
		return fmt.Errorf("failed to update limit_counter: %v", err)
	}

	if updatedUserDoc.LimitCounter.PipelineBuilds > subscriptionDoc.PipelineSpecs.DailyBuilds {
		return fmt.Errorf("subscription limit reached; no further pipeline builds can be performed")
	}

	return nil
}

type CheckDeploySubscriptionLimitInput struct {
	UserTable         string
	SubscriptionTable string
	UserDoc           user.User
}

// CheckDeploySubscriptionLimit updates the users limit_counter and checks if more pipeline_deployments can be performed.
// If the limit_counter values have expired, the pipeline_deployments are reset and the expiration time is set to the next day.
func CheckDeploySubscriptionLimit(transportCtx context.Context, dynamoClient *dynamodb.Client, input *CheckBuildSubscriptionLimitInput) error {
	subscriptionDoc, err := database.GetSingle[subscription.Subscription](transportCtx, dynamoClient, &database.GetSingleInput{
		Table: aws.String(input.SubscriptionTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: input.UserDoc.SubscriptionId},
		},
		ConditionExpr: aws.String("id = :id"),
	})
	if err != nil {
		return fmt.Errorf("failed to fetch subscription: %v", err)
	}

	// Update pipeline_deployments if the current counter has expired.
	_, err = database.UpdateSingle[user.User](transportCtx, dynamoClient, &database.UpdateSingleInput{
		Table: aws.String(input.UserTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: input.UserDoc.Id},
		},
		Upsert:    false,
		ReturnOld: false,
		AttributeNames: map[string]string{
			"#limit_counter":            "limit_counter",
			"#pipeline_deployments":     "pipeline_deployments",
			"#pipeline_deployments_exp": "pipeline_deployments_exp",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":pipeline_deployments": &dynamodbtypes.AttributeValueMemberN{Value: "0"},
			":current_timestamp": &dynamodbtypes.AttributeValueMemberN{Value: strconv.Itoa(
				int(time.Now().Unix()),
			)},
			":next_timestamp": &dynamodbtypes.AttributeValueMemberN{Value: strconv.Itoa(
				int(time.Now().Add(24 * time.Hour).Unix()),
			)},
		},
		ConditionExpr: aws.String("#limit_counter.#pipeline_deployments_exp < :current_timestamp"),
		UpdateExpr:    aws.String("SET #limit_counter.#pipeline_deployments = :pipeline_deployments, #limit_counter.#pipeline_deployments_exp = :next_timestamp"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); !ok {
			return fmt.Errorf("failed to reset limit_counter: %v", err)
		}
	}

	updatedUserDoc, err := database.UpdateSingle[user.User](transportCtx, dynamoClient, &database.UpdateSingleInput{
		Table: aws.String(input.UserTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: input.UserDoc.Id},
		},
		Upsert:    false,
		ReturnOld: false,
		AttributeNames: map[string]string{
			"#limit_counter":        "limit_counter",
			"#pipeline_deployments": "pipeline_deployments",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":increment": &dynamodbtypes.AttributeValueMemberN{Value: "1"},
		},
		UpdateExpr: aws.String("ADD #limit_counter.#pipeline_deployments :increment"),
	})
	if err != nil {
		return fmt.Errorf("failed to update limit_counter: %v", err)
	}

	if updatedUserDoc.LimitCounter.PipelineDeployments > subscriptionDoc.PipelineSpecs.DailyDeployments {
		return fmt.Errorf("subscription limit reached; no further pipeline deployments can be performed")
	}

	return nil
}
