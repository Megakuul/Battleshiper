package updateproject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/user"
)

type repositoryInput struct {
	Id     int64  `json:"id"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

type updateProjectInput struct {
	ProjectName     string          `json:"project_name"`
	BuildCommand    string          `json:"build_command"`
	OutputDirectory string          `json:"output_directory"`
	Repository      repositoryInput `json:"repository"`
}

type updateProjectOutput struct {
	Message string `json:"message"`
}

// HandleUpdateProject updates specified project fields.
func HandleUpdateProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleUpdateProject(request, transportCtx, routeCtx)
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

func runHandleUpdateProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*updateProjectOutput, int, error) {
	var updateProjectInput updateProjectInput
	err := json.Unmarshal([]byte(request.Body), &updateProjectInput)
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
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load user record from database")
	}

	updateSpec := map[string]dynamodbtypes.AttributeValue{}
	if updateProjectInput.BuildCommand != "" {
		updateSpec["build_command"] = &dynamodbtypes.AttributeValueMemberS{
			Value: updateProjectInput.BuildCommand,
		}
	}
	if updateProjectInput.OutputDirectory != "" {
		updateSpec["output_directory"] = &dynamodbtypes.AttributeValueMemberS{
			Value: updateProjectInput.OutputDirectory,
		}
	}
	if updateProjectInput.Repository.Id != 0 {
		repositoryAttributes, err := attributevalue.Marshal(&project.Repository{
			Id:     updateProjectInput.Repository.Id,
			URL:    updateProjectInput.Repository.URL,
			Branch: updateProjectInput.Repository.Branch,
		})
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to serialize repository")
		}
		updateSpec["repository"] = repositoryAttributes
	}

	updateAttributeNames, updateAttributeValues, updateExpression := constructFromSpec(updateSpec)

	updateAttributeNames["#owner_id"] = "owner_id"
	updateAttributeNames["#deleted"] = "deleted"
	updateAttributeValues[":owner_id"] = &dynamodbtypes.AttributeValueMemberS{Value: userDoc.Id}
	updateAttributeValues[":deleted"] = &dynamodbtypes.AttributeValueMemberBOOL{Value: false}

	_, err = database.UpdateSingle[project.Project](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
		Table: aws.String(routeCtx.ProjectTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"name": &dynamodbtypes.AttributeValueMemberS{Value: updateProjectInput.ProjectName},
		},
		AttributeNames:  updateAttributeNames,
		AttributeValues: updateAttributeValues,
		ConditionExpr:   aws.String("#owner_id = :owner_id AND #deleted = :deleted"),
		UpdateExpr:      aws.String(updateExpression),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("project not found")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load project from database")
	}

	return &updateProjectOutput{
		Message: "project updated",
	}, http.StatusOK, nil
}

// constructFromSpec converts a updateSpec map into attributeNames, attributeValues and a updateExpression.
func constructFromSpec(updateSpec map[string]dynamodbtypes.AttributeValue) (map[string]string, map[string]dynamodbtypes.AttributeValue, string) {
	var (
		attributeNames   = map[string]string{}
		attributeValues  = map[string]dynamodbtypes.AttributeValue{}
		updateExpression = ""
	)
	for key, value := range updateSpec {
		attributeNames[fmt.Sprintf("#%s", key)] = key
		attributeValues[fmt.Sprintf(":%s", key)] = value
		if updateExpression == "" {
			updateExpression = "SET "
		} else {
			updateExpression += ","
		}
		updateExpression += fmt.Sprintf("#%s = :%s", key, key)
	}
	return attributeNames, attributeValues, updateExpression
}
