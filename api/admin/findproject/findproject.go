package findproject

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/api/admin/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
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

	userDoc, err := database.GetSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: routeCtx.UserTable,
		Index: "",
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: userToken.Id},
		},
		ConditionExpr: "id = :id",
	})
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to load user record from database")
	}

	if !rbac.CheckPermission(userDoc.Roles, rbac.READ_PROJECT) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	var foundProjectDocs []project.Project
	if findProjectInput.OwnerId != "" {
		foundProjectDocs, err = database.GetMany[project.Project](transportCtx, routeCtx.DynamoClient, &database.GetManyInput{
			Table: routeCtx.ProjectTable,
			Index: project.GSI_OWNER_ID,
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":owner_id": &dynamodbtypes.AttributeValueMemberS{Value: findProjectInput.OwnerId},
			},
			ConditionExpr: "owner_id = :owner_id",
		})
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("failed load projects on database")
		}
	} else if findProjectInput.ProjectName != "" {
		foundProjectDocs, err = database.GetMany[project.Project](transportCtx, routeCtx.DynamoClient, &database.GetManyInput{
			Table: routeCtx.ProjectTable,
			Index: "",
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":name": &dynamodbtypes.AttributeValueMemberS{Value: findProjectInput.ProjectName},
			},
			ConditionExpr: "name = :name",
		})
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("failed load projects on database")
		}
	} else {
		return nil, http.StatusBadRequest, fmt.Errorf("specify at least one query option")
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
