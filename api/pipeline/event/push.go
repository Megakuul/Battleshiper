package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/go-playground/webhooks/v6/github"
	"github.com/google/uuid"
	"github.com/megakuul/battleshiper/api/pipeline/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/user"
)

func handleRepoPush(transportCtx context.Context, routeCtx routecontext.Context, event github.PushPayload) (int, error) {
	// events not triggered by branch pushes are not handled.
	if !strings.HasPrefix(event.Ref, "refs/heads/") {
		return http.StatusOK, nil
	}
	branch := strings.TrimPrefix(event.Ref, "refs/heads/")

	userDoc, err := database.GetSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: aws.String(routeCtx.UserTable),
		Index: aws.String(user.GSI_INSTALLATION_ID),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":installation_id": &dynamodbtypes.AttributeValueMemberN{Value: strconv.Itoa(event.Installation.ID)},
		},
		ConditionExpr: aws.String("installation_id = :installation_id"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return http.StatusNotFound, fmt.Errorf("user not found")
		}
		return http.StatusInternalServerError, fmt.Errorf("failed to load user record from database")
	}

	foundProjectDocs, err := database.GetMany[project.Project](transportCtx, routeCtx.DynamoClient, &database.GetManyInput{
		Table: aws.String(routeCtx.ProjectTable),
		Index: aws.String(project.GSI_OWNER_ID),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":owner_id": &dynamodbtypes.AttributeValueMemberS{Value: userDoc.Id},
		},
		ConditionExpr: aws.String("owner_id = :owner_id"),
	})
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to load projects from database")
	}

	for _, projectDoc := range foundProjectDocs {
		if projectDoc.Repository.Id != event.Repository.ID {
			continue
		}
		if projectDoc.Repository.Branch != branch {
			continue
		}
		if !projectDoc.Initialized || projectDoc.Deleted {
			continue
		}

		execId := uuid.New().String()
		eventResult := project.EventResult{
			ExecutionIdentifier: execId,
		}
		if err = initiateProjectBuild(transportCtx, routeCtx, execId, userDoc, &projectDoc); err != nil {
			eventResult.Successful = false
			eventResult.Timepoint = time.Now().Unix()

			eventResultAttributes, err := attributevalue.Marshal(&eventResult)
			if err != nil {
				return http.StatusInternalServerError, fmt.Errorf("failed to serialize eventresult")
			}

			_, err = database.UpdateSingle[project.Project](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
				Table: aws.String(routeCtx.ProjectTable),
				PrimaryKey: map[string]dynamodbtypes.AttributeValue{
					"name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.Name},
				},
				AttributeNames: map[string]string{
					"#last_event_result": "last_event_result",
					"#status":            "status",
				},
				AttributeValues: map[string]dynamodbtypes.AttributeValue{
					":last_event_result": eventResultAttributes,
					":status":            &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("EVENT FAILED: %v", err)},
				},
				UpdateExpr: aws.String("SET #last_event_result = :last_event_result, #status = :status"),
			})
			if err != nil {
				return http.StatusInternalServerError, fmt.Errorf("failed to update project: %v", err)
			}
			return http.StatusInternalServerError, fmt.Errorf("failed to initiate project build: %v", err)
		} else {
			eventResult.Successful = true
			eventResult.Timepoint = time.Now().Unix()

			eventResultAttributes, err := attributevalue.Marshal(&eventResult)
			if err != nil {
				return http.StatusInternalServerError, fmt.Errorf("failed to serialize eventresult")
			}

			_, err = database.UpdateSingle[project.Project](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
				Table: aws.String(routeCtx.ProjectTable),
				PrimaryKey: map[string]dynamodbtypes.AttributeValue{
					"name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.Name},
				},
				AttributeNames: map[string]string{
					"#last_event_result": "last_event_result",
				},
				AttributeValues: map[string]dynamodbtypes.AttributeValue{
					":last_event_result": eventResultAttributes,
				},
				UpdateExpr: aws.String("SET #last_event_result = :last_event_result"),
			})
			if err != nil {
				return http.StatusInternalServerError, fmt.Errorf("failed to update project: %v", err)
			}
		}
	}

	return http.StatusOK, nil
}

func initiateProjectBuild(transportCtx context.Context, routeCtx routecontext.Context, execId string, userDoc *user.User, projectDoc *project.Project) error {
	cloudLogger, err := pipeline.NewCloudLogger(transportCtx, routeCtx.CloudwatchClient, projectDoc.DedicatedInfrastructure.EventLogGroup, execId)
	if err != nil {
		return err
	}

	cloudLogger.WriteLog("START INIT %s", execId)
	cloudLogger.WriteLog("Event triggered by github webhook")
	if err := cloudLogger.PushLogs(); err != nil {
		return err
	}

	cloudLogger.WriteLog("Generating installation token...")
	installToken, _, err := routeCtx.GithubAppClient.Apps.CreateInstallationToken(transportCtx, userDoc.InstallationId, nil)
	if err != nil {
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return fmt.Errorf("failed to generate installation token: %v", err)
	}

	cloudLogger.WriteLog("Emitting event to pipeline...")
	err = emitBuildEvent(transportCtx, routeCtx, execId, installToken.GetToken(), userDoc, projectDoc)
	if err != nil {
		cloudLogger.WriteLog("failed to emit build event: %v", err)
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return fmt.Errorf("failed to emit build event: %v", err)
	} else {
		cloudLogger.WriteLog("project build was successfully initiated")
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
	}

	return nil
}

func emitBuildEvent(transportCtx context.Context, routeCtx routecontext.Context, execId, installToken string, userDoc *user.User, projectDoc *project.Project) error {
	err := pipeline.CheckBuildSubscriptionLimit(transportCtx, routeCtx.DynamoClient, &pipeline.CheckBuildSubscriptionLimitInput{
		UserTable:         routeCtx.UserTable,
		SubscriptionTable: routeCtx.SubscriptionTable,
		UserDoc:           *userDoc,
	})
	if err != nil {
		return err
	}

	deployTicket, err := pipeline.CreateTicket(routeCtx.DeployTicketOptions, userDoc.Id, projectDoc.Name)
	if err != nil {
		return fmt.Errorf("failed to create pipeline ticket")
	}

	// Usage of the installation token like this is documented here:
	// https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-as-a-github-app-installation
	tokenRepositoryUrl := fmt.Sprintf(
		"https://x-access-token:%s@%s",
		installToken,
		strings.TrimPrefix(projectDoc.Repository.URL, "https://"),
	)

	buildRequest := &event.BuildRequest{
		ExecutionIdentifier: execId,
		DeployTicket:        deployTicket,
		RepositoryURL:       tokenRepositoryUrl,
		RepositoryBranch:    projectDoc.Repository.Branch,
		BuildCommand:        projectDoc.BuildCommand,
		OutputDirectory:     projectDoc.OutputDirectory,
	}
	buildRequestRaw, err := json.Marshal(buildRequest)
	if err != nil {
		return fmt.Errorf("failed to serialize build request")
	}

	eventEntry := eventbridgetypes.PutEventsRequestEntry{
		Source:       aws.String(routeCtx.BuildEventOptions.Source),
		DetailType:   aws.String(routeCtx.BuildEventOptions.Action),
		Detail:       aws.String(string(buildRequestRaw)),
		EventBusName: aws.String(routeCtx.BuildEventOptions.EventBus),
	}
	res, err := routeCtx.EventClient.PutEvents(transportCtx, &eventbridge.PutEventsInput{
		Entries: []eventbridgetypes.PutEventsRequestEntry{eventEntry},
	})
	if err != nil || res.FailedEntryCount > 0 {
		return fmt.Errorf("failed to emit build event to the pipeline")
	}

	return nil
}
