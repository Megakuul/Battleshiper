package createproject

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
)

const MIN_PROJECT_NAME_CHARACTERS = 3

type repositoryInput struct {
	Id     int64  `json:"id"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

type createProjectInput struct {
	ProjectName  string          `json:"project_name"`
	BuildCommand string          `json:"build_command"`
	Repository   repositoryInput `json:"repository"`
}

type createProjectOutput struct {
	Message string `json:"message"`
}

// HandleCreateProject creates a project.
func HandleCreateProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleCreateProject(request, transportCtx, routeCtx)
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

func runHandleCreateProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*createProjectOutput, int, error) {
	var createProjectInput createProjectInput
	err := json.Unmarshal([]byte(request.Body), &createProjectInput)
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

	subscriptionCollection := routeCtx.Database.Collection(subscription.SUBSCRIPTION_COLLECTION)

	subscriptionDoc := &subscription.Subscription{}
	err = subscriptionCollection.FindOne(transportCtx, bson.M{"id": userDoc.SubscriptionId}).Decode(&subscriptionDoc)
	if err == mongo.ErrNoDocuments {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have a valid subscription associated")
	} else if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch subscription from database")
	}

	projectCollection := routeCtx.Database.Collection(project.PROJECT_COLLECTION)

	count, err := projectCollection.CountDocuments(transportCtx, bson.M{
		"owner_id": userDoc.Id,
		"deleted":  false,
	})
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch projects from database")
	}

	if int(count) >= subscriptionDoc.Projects {
		return nil, http.StatusForbidden, fmt.Errorf("subscription limit reached; no additional projects can be created")
	}

	if len(createProjectInput.ProjectName) <= MIN_PROJECT_NAME_CHARACTERS {
		return nil, http.StatusBadRequest, fmt.Errorf("project name must contain at least %d characters", MIN_PROJECT_NAME_CHARACTERS)
	}

	initTicket, err := pipeline.CreateTicket(routeCtx.InitTicketOptions, userDoc.Id, createProjectInput.ProjectName)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create pipeline ticket")
	}

	_, err = projectCollection.InsertOne(transportCtx, project.Project{
		Name:        createProjectInput.ProjectName,
		OwnerId:     userDoc.Id,
		Deleted:     false,
		Initialized: false,
		Repository: project.Repository{
			Id:     createProjectInput.Repository.Id,
			URL:    createProjectInput.Repository.URL,
			Branch: createProjectInput.Repository.Branch,
		},
		BuildCommand: createProjectInput.BuildCommand,

		InfrastructureStackId: "",
		ApiRoutePath:          "",
		StaticBucketPath:      "",
		FunctionBucketPath:    "",
		BuildAssetBucketPath:  "",
		LogGroup:              "",
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, http.StatusConflict, fmt.Errorf("project name is already registered")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to insert project to database")
	}

	initRequest := &event.InitRequest{
		InitTicket:  initTicket,
		ProjectName: createProjectInput.ProjectName,
	}
	initRequestRaw, err := json.Marshal(initRequest)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to serialize init request")
	}

	eventEntry := types.PutEventsRequestEntry{
		Source:       aws.String("ch.megakuul.battleshiper"),
		DetailType:   aws.String("battleshiper.init"),
		Detail:       aws.String(string(initRequestRaw)),
		EventBusName: aws.String(routeCtx.EventBus),
	}
	res, err := routeCtx.EventClient.PutEvents(transportCtx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{eventEntry},
	})
	if err != nil || res.FailedEntryCount > 0 {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to emit init event")
	}

	return &createProjectOutput{
		Message: "project created; project infrastructure is being initialized...",
	}, http.StatusOK, nil
}
