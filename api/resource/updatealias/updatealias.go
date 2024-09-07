package updatealias

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	cloudfrontkeyvaluetypes "github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/user"
)

const (
	MAX_ALIAS_SIZE = 30
)

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

	userCollection := routeCtx.Database.Collection(user.USER_COLLECTION)

	userDoc := &user.User{}
	err = userCollection.FindOne(transportCtx, bson.M{"id": userToken.Id}).Decode(&userDoc)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to load user record from database")
	}

	projectCollection := routeCtx.Database.Collection(project.PROJECT_COLLECTION)

	projectDoc := &project.Project{}
	err = projectCollection.FindOne(transportCtx, bson.D{
		{Key: "name", Value: updateAliasInput.ProjectName},
		{Key: "owner_id", Value: userDoc.Id},
	}).Decode(&projectDoc)
	if err == mongo.ErrNoDocuments {
		return nil, http.StatusNotFound, fmt.Errorf("project does not exist")
	} else if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update project on database")
	}

	if err := validateAliases(projectDoc.Name, updateAliasInput.Aliases); err != nil {
		return nil, http.StatusBadRequest, err
	}

	if err := updateAliases(transportCtx, routeCtx, projectDoc.Name, projectDoc.Aliases, updateAliasInput.Aliases); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
		"$set": bson.M{
			"aliases": updateAliasInput.Aliases,
		},
	})
	if err != nil || result.MatchedCount < 1 {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update project on database")
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
