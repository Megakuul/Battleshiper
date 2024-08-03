package deleteproject

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/project"
)

type deleteProjectInput struct {
	ProjectId string `json:"project_id"`
}

type deleteProjectOutput struct {
	Message string `json:"message"`
}

// HandleDeleteProject marks the specified project as deleted.
func HandleDeleteProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleDeleteProject(request, transportCtx, routeCtx)
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

func runHandleDeleteProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*deleteProjectOutput, int, error) {

	var deleteProjectInput deleteProjectInput
	err := json.Unmarshal([]byte(request.Body), &deleteProjectInput)
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

	projectCollection := routeCtx.Database.Collection(project.PROJECT_COLLECTION)

	deletedProject := &project.Project{}
	err = projectCollection.FindOneAndUpdate(transportCtx, bson.M{"id": deleteProjectInput.ProjectId}, bson.M{
		"$set": bson.M{
			"deleted": bson.M{
				"$cond": bson.M{
					"if":   bson.M{"$eq": bson.A{"$owner_id", userToken.Id}},
					"then": true,
					"else": false,
				},
			},
		},
	}).Decode(&deletedProject)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to mark project as deleted on database")
	}

	if !deletedProject.Deleted {
		return nil, http.StatusForbidden, fmt.Errorf("user is not the owner of the project")
	}

	return &deleteProjectOutput{
		Message: "successfully marked project as deleted",
	}, http.StatusOK, nil
}
