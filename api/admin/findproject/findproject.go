package findproject

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
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/user"
)

var logger = log.New(os.Stderr, "ADMIN FINDPROJECT: ", 0)

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
	OwnerId := request.QueryStringParameters["owner_id"]
	ProjectName := request.QueryStringParameters["project_name"]

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

	if !rbac.CheckPermission(userDoc.Roles, rbac.READ_PROJECT) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	var foundProjectDocs []project.Project
	if OwnerId != "" {
		foundProjectDocs, err = database.GetMany[project.Project](transportCtx, routeCtx.DynamoClient, &database.GetManyInput{
			Table: aws.String(routeCtx.ProjectTable),
			Index: aws.String(project.GSI_OWNER_ID),
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":owner_id": &dynamodbtypes.AttributeValueMemberS{Value: OwnerId},
			},
			ConditionExpr: aws.String("owner_id = :owner_id"),
		})
		if err != nil {
			logger.Printf("failed load projects on database: %v\n", err)
			return nil, http.StatusInternalServerError, fmt.Errorf("failed load projects on database")
		}
	} else if ProjectName != "" {
		foundProjectDocs, err = database.GetMany[project.Project](transportCtx, routeCtx.DynamoClient, &database.GetManyInput{
			Table: aws.String(routeCtx.ProjectTable),
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":project_name": &dynamodbtypes.AttributeValueMemberS{Value: ProjectName},
			},
			ConditionExpr: aws.String("project_name = :project_name"),
		})
		if err != nil {
			logger.Printf("failed load projects from database: %v\n", err)
			return nil, http.StatusInternalServerError, fmt.Errorf("failed load projects from database")
		}
	} else {
		return nil, http.StatusBadRequest, fmt.Errorf("specify at least one query option")
	}

	foundProjectOutput := []projectOutput{}
	for _, project := range foundProjectDocs {
		foundProjectOutput = append(foundProjectOutput, projectOutput{
			Name:        project.ProjectName,
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
