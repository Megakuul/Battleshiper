package event

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/webhooks/v6/github"
	"github.com/megakuul/battleshiper/api/pipeline/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/user"
)

func handleAppInstallation(transportCtx context.Context, routeCtx routecontext.Context, event github.InstallationPayload) (int, error) {
	userId := event.Installation.Account.ID

	installedRepos := map[int64]user.Repository{}
	for _, repo := range event.Repositories {
		installedRepos[repo.ID] = user.Repository{
			Id:       repo.ID,
			Name:     repo.Name,
			FullName: repo.FullName,
		}
	}

	repositories, err := attributevalue.Marshal(&installedRepos)
	if err != nil {
		logger.Printf("failed to serialize repositories: %v\n", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to serialize repositories")
	}

	_, err = database.UpdateSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
		Table: aws.String(routeCtx.UserTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: strconv.Itoa(int(userId))},
		},
		AttributeNames: map[string]string{
			"#installation_id": "installation_id",
			"#repositories":    "repositories",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":installation_id": &dynamodbtypes.AttributeValueMemberN{Value: strconv.Itoa(int(event.Installation.ID))},
			":repositories":    repositories,
		},
		UpdateExpr: aws.String("SET #installation_id = :installation_id, #repositories = :repositories"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return http.StatusNotFound, fmt.Errorf("user not found")
		}
		logger.Printf("failed to update user record on database: %v\n", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to update user record on database")
	}

	return http.StatusOK, nil
}
