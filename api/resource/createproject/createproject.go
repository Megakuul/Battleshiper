package createproject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	eventtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
)

const MIN_PROJECT_NAME_CHARACTERS = 3

var logger = log.New(os.Stderr, "RESOURCE CREATEPROJECT: ", 0)

type repositoryInput struct {
	Id     int64  `json:"id"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

type createProjectInput struct {
	ProjectName     string          `json:"project_name"`
	BuildImage      string          `json:"build_image"`
	BuildCommand    string          `json:"build_command"`
	OutputDirectory string          `json:"output_directory"`
	Repository      repositoryInput `json:"repository"`
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

	subscriptionDoc, err := database.GetSingle[subscription.Subscription](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: aws.String(routeCtx.SubscriptionTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: userDoc.SubscriptionId},
		},
		ConditionExpr: aws.String("id = :id"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusBadRequest, fmt.Errorf("user does not have a valid subscription associated")
		}
		logger.Printf("failed to load subscription from database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load subscription from database")
	}

	projectDocs, err := database.GetMany[project.Project](transportCtx, routeCtx.DynamoClient, &database.GetManyInput{
		Table: aws.String(routeCtx.ProjectTable),
		Index: aws.String(project.GSI_OWNER_ID),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":owner_id": &dynamodbtypes.AttributeValueMemberS{Value: userDoc.Id},
		},
		ConditionExpr: aws.String("owner_id = :owner_id"),
	})
	if err != nil {
		logger.Printf("failed to count projects on database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to count projects on database")
	}

	if len(projectDocs) >= int(subscriptionDoc.ProjectSpecs.ProjectCount) {
		return nil, http.StatusForbidden, fmt.Errorf("subscription limit reached; no additional projects can be created")
	}

	createProjectInput.ProjectName = strings.ToLower(createProjectInput.ProjectName)

	// not covered by the regex because it is important that the project name is NOT "" which would be unexpected.
	// I don't want to rely on a regex that I can't understand when I skim through it.
	if len(createProjectInput.ProjectName) <= MIN_PROJECT_NAME_CHARACTERS {
		return nil, http.StatusBadRequest, fmt.Errorf("project name must contain at least %d characters", MIN_PROJECT_NAME_CHARACTERS)
	}

	reg := regexp.MustCompile("^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$")
	if !reg.MatchString(createProjectInput.ProjectName) {
		return nil, http.StatusBadRequest, fmt.Errorf("project name must match a valid domain fragment format")
	}

	err = database.PutSingle(transportCtx, routeCtx.DynamoClient, &database.PutSingleInput[project.Project]{
		Table: aws.String(routeCtx.ProjectTable),
		Item: project.Project{
			Name:         createProjectInput.ProjectName,
			OwnerId:      userDoc.Id,
			Deleted:      false,
			Initialized:  false,
			Status:       "",
			Aliases:      map[string]struct{}{createProjectInput.ProjectName: {}},
			PipelineLock: true,
			Repository: project.Repository{
				Id:     createProjectInput.Repository.Id,
				URL:    createProjectInput.Repository.URL,
				Branch: createProjectInput.Repository.Branch,
			},
			BuildImage:      createProjectInput.BuildImage,
			BuildCommand:    createProjectInput.BuildCommand,
			OutputDirectory: createProjectInput.OutputDirectory,
		},
		ProtectionAttributeName: aws.String("name"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusBadRequest, fmt.Errorf("project name is already registered")
		}
		logger.Printf("failed to insert project to database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to insert project to database")
	}

	if err := initAlias(transportCtx, routeCtx, createProjectInput.ProjectName); err != nil {
		logger.Printf("%v\n", err)
		return nil, http.StatusInternalServerError, err
	}

	initTicket, err := pipeline.CreateTicket(routeCtx.InitEventOptions.TicketOpts, userDoc.Id, createProjectInput.ProjectName)
	if err != nil {
		logger.Printf("failed to create pipeline ticket: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create pipeline ticket")
	}
	initRequest := &event.InitRequest{
		InitTicket: initTicket,
	}
	initRequestRaw, err := json.Marshal(initRequest)
	if err != nil {
		logger.Printf("failed to serialize init request: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to serialize init request")
	}
	eventEntry := eventtypes.PutEventsRequestEntry{
		Source:       aws.String(routeCtx.InitEventOptions.Source),
		DetailType:   aws.String(routeCtx.InitEventOptions.Action),
		Detail:       aws.String(string(initRequestRaw)),
		EventBusName: aws.String(routeCtx.InitEventOptions.EventBus),
	}
	res, err := routeCtx.EventClient.PutEvents(transportCtx, &eventbridge.PutEventsInput{
		Entries: []eventtypes.PutEventsRequestEntry{eventEntry},
	})
	if err != nil || res.FailedEntryCount > 0 {
		logger.Printf("failed to emit init event: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to emit init event")
	}

	return &createProjectOutput{
		Message: "project created; project infrastructure is being initialized...",
	}, http.StatusOK, nil
}

// initAlias uploads the initial alias to the cloudfront cache.
func initAlias(transportCtx context.Context, routeCtx routecontext.Context, projectName string) error {
	storeMetadata, err := routeCtx.CloudfrontCacheClient.DescribeKeyValueStore(transportCtx, &cloudfrontkeyvaluestore.DescribeKeyValueStoreInput{
		KvsARN: aws.String(routeCtx.CloudfrontCacheArn),
	})
	if err != nil {
		return fmt.Errorf("failed to describe cdn store: %v", err)
	}
	_, err = routeCtx.CloudfrontCacheClient.PutKey(transportCtx, &cloudfrontkeyvaluestore.PutKeyInput{
		KvsARN:  aws.String(routeCtx.CloudfrontCacheArn),
		Key:     aws.String(projectName),
		Value:   aws.String(projectName),
		IfMatch: storeMetadata.ETag,
	})
	if err != nil {
		return fmt.Errorf("failed to insert alias to cdn store: %v", err)
	}

	return nil
}
