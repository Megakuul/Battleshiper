package updatealias

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	cloudfrontkeyvaluetypes "github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore/types"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
)

const MAX_ALIAS_SIZE = 30

var logger = log.New(os.Stderr, "RESOURCE UPDATEALIAS: ", 0)

type updateAliasInput struct {
	ProjectName string              `json:"project_name"`
	Aliases     map[string]struct{} `json:"aliases"`
}

type updateAliasOutput struct {
	Message string `json:"message"`
}

// HandleUpdateAlias updates specified aliases on the cdn cache.
func HandleUpdateAlias(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleUpdateAlias(request, transportCtx, routeCtx)
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

func runHandleUpdateAlias(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*updateAliasOutput, int, error) {
	var updateAliasInput updateAliasInput
	err := json.Unmarshal([]byte(request.Body), &updateAliasInput)
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
		Table: aws.String(routeCtx.UserTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: userToken.Id},
		},
		ConditionExpr: aws.String("id = :id"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("user not found")
		}
		logger.Printf("failed to load user from database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load user from database")
	}

	projectDoc, err := database.GetSingle[project.Project](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: aws.String(routeCtx.ProjectTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":name":     &dynamodbtypes.AttributeValueMemberS{Value: updateAliasInput.ProjectName},
			":owner_id": &dynamodbtypes.AttributeValueMemberS{Value: userDoc.Id},
		},
		ConditionExpr: aws.String("name = :name AND owner_id = :owner_id"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("project not found")
		}
		logger.Printf("failed to load project from database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load project from database")
	}

	if err := validateAliases(projectDoc.Name, updateAliasInput.Aliases); err != nil {
		return nil, http.StatusBadRequest, err
	}

	subscriptionDoc, err := database.GetSingle[subscription.Subscription](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: aws.String(routeCtx.SubscriptionTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: userDoc.SubscriptionId},
		},
		ConditionExpr: aws.String("id = :id"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusBadRequest, fmt.Errorf("user does not have a valid subscription associated")
		}
		logger.Printf("failed to load subscription from database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load subscription from database")
	}

	if len(updateAliasInput.Aliases) > int(subscriptionDoc.ProjectSpecs.AliasCount) {
		return nil, http.StatusBadRequest, fmt.Errorf("subscription limit reached; no additional aliases can be created")
	}

	if err := updateAliases(transportCtx, routeCtx, projectDoc.Name, projectDoc.Aliases, updateAliasInput.Aliases); err != nil {
		logger.Printf("%v\n", err)
		return nil, http.StatusInternalServerError, err
	}

	aliasAttributes, err := attributevalue.Marshal(&updateAliasInput.Aliases)
	if err != nil {
		logger.Printf("failed to serialize alias attributes: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to serialize alias attributes")
	}

	_, err = database.UpdateSingle[project.Project](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
		Table: aws.String(routeCtx.ProjectTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.Name},
		},
		AttributeNames: map[string]string{
			"#aliases": "aliases",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":aliases": aliasAttributes,
		},
		UpdateExpr: aws.String("#aliases = :aliases"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("project not found")
		}
		logger.Printf("failed to load project from database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load project from database")
	}

	return &updateAliasOutput{
		Message: "aliases updated",
	}, http.StatusOK, nil
}

// validateAliases checks if the aliases are valid.
func validateAliases(projectName string, aliases map[string]struct{}) error {
	expectedSuffix := fmt.Sprintf(".%s", projectName)

	for alias := range aliases {
		if len(alias) > MAX_ALIAS_SIZE {
			return fmt.Errorf("invalid alias: alias cannot be longer then %d", MAX_ALIAS_SIZE)
		}
		if !strings.HasSuffix(alias, expectedSuffix) && alias != projectName {
			return fmt.Errorf("invalid alias: alias must end with '%s'", expectedSuffix)
		}
	}

	return nil
}

// updateAliases merges the old and new aliases and uploads them to the cloudfront cache.
func updateAliases(transportCtx context.Context, routeCtx routecontext.Context, projectName string, oldAliases, newAliases map[string]struct{}) error {
	addAliasKeys := []cloudfrontkeyvaluetypes.PutKeyRequestListItem{}
	for alias := range newAliases {
		addAliasKeys = append(addAliasKeys, cloudfrontkeyvaluetypes.PutKeyRequestListItem{
			Key:   aws.String(alias),
			Value: aws.String(projectName),
		})
	}

	deleteAliasKeys := []cloudfrontkeyvaluetypes.DeleteKeyRequestListItem{}
	for alias := range oldAliases {
		if _, exists := newAliases[alias]; !exists {
			deleteAliasKeys = append(deleteAliasKeys, cloudfrontkeyvaluetypes.DeleteKeyRequestListItem{
				Key: aws.String(alias),
			})
		}
	}

	storeMetadata, err := routeCtx.CloudfrontCacheClient.DescribeKeyValueStore(transportCtx, &cloudfrontkeyvaluestore.DescribeKeyValueStoreInput{
		KvsARN: aws.String(routeCtx.CloudfrontCacheArn),
	})
	if err != nil {
		return fmt.Errorf("failed to describe cdn store: %v", err)
	}
	_, err = routeCtx.CloudfrontCacheClient.UpdateKeys(transportCtx, &cloudfrontkeyvaluestore.UpdateKeysInput{
		KvsARN:  aws.String(routeCtx.CloudfrontCacheArn),
		Puts:    addAliasKeys,
		Deletes: deleteAliasKeys,
		IfMatch: storeMetadata.ETag,
	})
	if err != nil {
		return fmt.Errorf("failed to update cdn store keys: %v", err)
	}

	return nil
}
