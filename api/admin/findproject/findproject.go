package findproject

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/megakuul/battleshiper/api/admin/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/user"
)

type findProjectInput struct {
	ProjectName string `json:"project_name"`
	OwnerId     string `json:"owner_id"`
}

type repositoryOutput struct {
	Id     int64  `json:"id"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

type projectOutput struct {
	Name        string              `json:"name"`
	Deleted     bool                `json:"deleted"`
	Initialized bool                `json:"initialized"`
	Status      string              `json:"status"`
	Aliases     map[string]struct{} `json:"aliases"`
	Repository  repositoryOutput    `json:"repository"`
	OwnerId     string              `json:"owner_id"`
}

type findProjectOutput struct {
	Message  string          `json:"message"`
	Projects []projectOutput `json:"projects"`
}

// HandleFindProject performs a lookup for the specified projects and returns them as json object.
func HandleFindProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleFindProject(request, transportCtx, routeCtx)
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

func runHandleFindProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*findProjectOutput, int, error) {
	var findProjectInput findProjectInput
	err := json.Unmarshal([]byte(request.Body), &findProjectInput)
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

	if !rbac.CheckPermission(userDoc.Roles, rbac.READ_PROJECT) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	projectCollection := routeCtx.Database.Collection(project.PROJECT_COLLECTION)

	cursor, err := projectCollection.Find(transportCtx,
		bson.M{"$or": bson.A{
			bson.M{"name": findProjectInput.ProjectName},
			bson.M{"owner_id": findProjectInput.OwnerId},
		}},
	)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch data from database")
	}

	foundProjectDocs := []project.Project{}
	err = cursor.All(transportCtx, &foundProjectDocs)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch and decode projects")
	}

	foundProjectOutput := []projectOutput{}
	for _, project := range foundProjectDocs {
		foundProjectOutput = append(foundProjectOutput, projectOutput{
			Name:        project.Name,
			Deleted:     project.Deleted,
			Initialized: project.Initialized,
			Status:      project.Status,
			Aliases:     project.Aliases,
			Repository: repositoryOutput{
				Id:     project.Repository.Id,
				URL:    project.Repository.URL,
				Branch: project.Repository.Branch,
			},
			OwnerId: project.OwnerId,
		})
	}

	return &findProjectOutput{
		Message:  "projects fetched",
		Projects: foundProjectOutput,
	}, http.StatusOK, nil
}
