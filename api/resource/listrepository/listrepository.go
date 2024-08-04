package listrepository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/go-github/github"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/oauth2"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/user"
)

type repositoryOutput struct {
	Id            int64  `json:"id"`
	FullName      string `json:"full_name"`
	URL           string `json:"url"`
	DefaultBranch string `json:"default_branch"`
}

type listRepositoryOutput struct {
	Message      string             `json:"message"`
	Repositories []repositoryOutput `json:"repositories"`
}

// HandleListRepositories performs a lookup for the repository the user granted access to (via github app).
func HandleListRepositories(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleListRepositories(request, transportCtx, routeCtx)
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

func runHandleListRepositories(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*listRepositoryOutput, int, error) {

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
		return nil, http.StatusBadRequest, fmt.Errorf("failed to load user data from database")
	}

	if userDoc.GithubInstallationId < 1 {
		return nil, http.StatusUnauthorized, fmt.Errorf("user did not install github app")
	}

	installToken, _, err := routeCtx.GithubAppClient.Apps.CreateInstallationToken(transportCtx, userDoc.GithubInstallationId)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch github installation token")
	}

	githubUserClient := github.NewClient(oauth2.NewClient(transportCtx, oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: installToken.GetToken(),
		},
	)))

	repos, _, err := githubUserClient.Apps.ListRepos(transportCtx, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch repositories")
	}

	outputRepos := []repositoryOutput{}
	for _, repo := range repos {
		outputRepos = append(outputRepos, repositoryOutput{
			Id:            *repo.ID,
			FullName:      *repo.FullName,
			URL:           *repo.CloneURL,
			DefaultBranch: *repo.DefaultBranch,
		})
	}

	return &listRepositoryOutput{
		Message:      "fetched repositories",
		Repositories: outputRepos,
	}, http.StatusOK, nil
}
