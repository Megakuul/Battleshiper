package updateproject

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
	"github.com/megakuul/battleshiper/lib/model/user"
)

type repositoryInput struct {
	Id     int64  `json:"id"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

type updateProjectInput struct {
	ProjectName     string          `json:"project_name"`
	BuildImage      string          `json:"build_image"`
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

	userCollection := routeCtx.Database.Collection(user.USER_COLLECTION)

	userDoc := &user.User{}
	err = userCollection.FindOne(transportCtx, bson.M{"id": userToken.Id}).Decode(&userDoc)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to load user record from database")
	}

	updateSpec := bson.M{}
	if updateProjectInput.BuildImage != "" {
		updateSpec["build_image"] = updateProjectInput.BuildImage
	}
	if updateProjectInput.BuildCommand != "" {
		updateSpec["build_command"] = updateProjectInput.BuildCommand
	}
	if updateProjectInput.OutputDirectory != "" {
		updateSpec["output_directory"] = updateProjectInput.OutputDirectory
	}
	if updateProjectInput.Repository.Id != 0 {
		updateSpec["repository"] = project.Repository{
			Id:     updateProjectInput.Repository.Id,
			URL:    updateProjectInput.Repository.URL,
			Branch: updateProjectInput.Repository.Branch,
		}
	}

	projectCollection := routeCtx.Database.Collection(project.PROJECT_COLLECTION)

	result, err := projectCollection.UpdateOne(transportCtx, bson.D{
		{Key: "name", Value: updateProjectInput.ProjectName},
		{Key: "owner_id", Value: userDoc.Id},
	}, bson.M{
		"$set": updateSpec,
	})
	if result.MatchedCount < 1 {
		return nil, http.StatusNotFound, fmt.Errorf("project does not exist")
	} else if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update project on database")
	}

	return &updateProjectOutput{
		Message: "project updated",
	}, http.StatusOK, nil
}
