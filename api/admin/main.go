package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
	"github.com/megakuul/battleshiper/lib/router"

	"github.com/megakuul/battleshiper/api/admin/deleteproject"
	"github.com/megakuul/battleshiper/api/admin/deleteuser"
	"github.com/megakuul/battleshiper/api/admin/findproject"
	"github.com/megakuul/battleshiper/api/admin/finduser"
	"github.com/megakuul/battleshiper/api/admin/routecontext"
	"github.com/megakuul/battleshiper/api/admin/updaterole"
	"github.com/megakuul/battleshiper/api/admin/updateuser"
	"github.com/megakuul/battleshiper/api/admin/upsertsubscription"
)

var (
	REGION              = os.Getenv("AWS_REGION")
	JWT_CREDENTIAL_ARN  = os.Getenv("JWT_CREDENTIAL_ARN")
	DATABASE_ENDPOINT   = os.Getenv("DATABASE_ENDPOINT")
	DATABASE_NAME       = os.Getenv("DATABASE_NAME")
	DATABASE_SECRET_ARN = os.Getenv("DATABASE_SECRET_ARN")
)

func main() {
	if err := run(); err != nil {
		log.Printf("ERROR INITIALIZATION: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	awsConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(REGION))
	if err != nil {
		return fmt.Errorf("failed to load aws config: %v", err)
	}

	databaseOptions, err := database.CreateDatabaseOptions(awsConfig, context.TODO(), DATABASE_SECRET_ARN, DATABASE_ENDPOINT, DATABASE_NAME)
	if err != nil {
		return err
	}
	databaseClient, err := mongo.Connect(context.TODO(), databaseOptions)
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		if err = databaseClient.Disconnect(ctx); err != nil {
			log.Printf("ERROR CLEANUP: %v\n", err)
		}
		cancel()
	}()
	databaseHandle := databaseClient.Database(DATABASE_NAME)

	database.SetupIndexes(databaseHandle.Collection(user.USER_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"id"}, SortingOrder: 1, Unique: true},
		{FieldNames: []string{"subscription_id"}, SortingOrder: 1, Unique: false},
	})

	database.SetupIndexes(databaseHandle.Collection(subscription.SUBSCRIPTION_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"id"}, SortingOrder: 1, Unique: true},
		{FieldNames: []string{"name"}, SortingOrder: 1, Unique: false},
	})

	database.SetupIndexes(databaseHandle.Collection(project.PROJECT_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"name"}, SortingOrder: 1, Unique: true},
		{FieldNames: []string{"owner_id"}, SortingOrder: 1, Unique: false},
	})

	jwtOptions, err := auth.CreateJwtOptions(awsConfig, context.TODO(), JWT_CREDENTIAL_ARN, 0)
	if err != nil {
		return err
	}

	httpRouter := router.NewRouter(routecontext.Context{
		JwtOptions: jwtOptions,
		Database:   databaseHandle,
	})

	httpRouter.AddRoute("GET", "/api/admin/finduser", finduser.HandleFindUser)
	httpRouter.AddRoute("GET", "/api/admin/findproject", findproject.HandleFindProject)
	httpRouter.AddRoute("PATCH", "/api/admin/updateuser", updateuser.HandleUpdateUser)
	httpRouter.AddRoute("PATCH", "/api/admin/updaterole", updaterole.HandleUpdateRole)
	httpRouter.AddRoute("PUT", "/api/admin/upsertsubscription", upsertsubscription.HandleUpsertSubscription)
	httpRouter.AddRoute("DELETE", "/api/admin/deleteuser", deleteuser.HandleDeleteUser)
	httpRouter.AddRoute("DELETE", "/api/admin/deleteproject", deleteproject.HandleDeleteProject)

	lambda.Start(httpRouter.Route)

	return nil
}
