package updateuser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/api/admin/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/user"
)

var logger = log.New(os.Stderr, "ADMIN UPDATEUSER: ", 0)

type updateInput struct {
	SubscriptionId string `json:"subscription_id"`
}

type updateUserInput struct {
	UserId string      `json:"user_id"`
	Update updateInput `json:"update"`
}

type updateUserOutput struct {
	Message string `json:"message"`
}

// HandleUpdateUser updates specified fields on a user identified by id.
func HandleUpdateUser(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleUpdateUser(request, transportCtx, routeCtx)
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

func runHandleUpdateUser(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*updateUserOutput, int, error) {
	var updateUserInput updateUserInput
	err := json.Unmarshal([]byte(request.Body), &updateUserInput)
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
		logger.Printf("failed to load user record from database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load user record from database")
	}

	if !rbac.CheckPermission(userDoc.Roles, rbac.WRITE_USER) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	_, err = database.UpdateSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
		Table: aws.String(routeCtx.UserTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: updateUserInput.UserId},
		},
		AttributeNames: map[string]string{
			"#subscription_id": "subscription_id",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":subscription_id": &dynamodbtypes.AttributeValueMemberS{Value: updateUserInput.Update.SubscriptionId},
		},
		UpdateExpr: aws.String("SET #subscription_id = :subscription_id"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("user to update was not found")
		}
		logger.Printf("failed to load user record from database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load user record from database")
	}

	return &updateUserOutput{
		Message: "user updated",
	}, http.StatusOK, nil
}
