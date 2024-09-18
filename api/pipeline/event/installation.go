package event

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-playground/webhooks/v6/github"
	"github.com/megakuul/battleshiper/api/pipeline/routecontext"
	"github.com/megakuul/battleshiper/lib/model/user"
	"go.mongodb.org/mongo-driver/bson"
)

func handleAppInstallation(transportCtx context.Context, routeCtx routecontext.Context, event github.InstallationPayload) (int, error) {
	userId := event.Installation.Account.ID

	installedRepos := []user.Repository{}
	for _, repo := range event.Repositories {
		installedRepos = append(installedRepos, user.Repository{
			Id:       repo.ID,
			Name:     repo.Name,
			FullName: repo.FullName,
		})
	}

	// MIG: Possible with update item and primary key
	result, err := userCollection.UpdateOne(transportCtx, bson.M{"id": userId}, bson.M{
		"$set": bson.M{
			"github_data": user.GithubData{
				InstallationId: event.Installation.ID,
				Repositories:   installedRepos,
			},
		},
	})
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to update user on the database")
	}

	if result.MatchedCount < 1 {
		return http.StatusNotFound, fmt.Errorf("user not found")
	}

	return http.StatusOK, nil
}
