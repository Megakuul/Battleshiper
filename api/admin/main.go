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

	"github.com/megakuul/battleshiper/api/user/info"
	"github.com/megakuul/battleshiper/api/user/register"
	"github.com/megakuul/battleshiper/api/user/routecontext"
)

var (
	REGION                = os.Getenv("AWS_REGION")
	JWT_CREDENTIAL_ARN    = os.Getenv("JWT_CREDENTIAL_ARN")
	REDIRECT_URI          = os.Getenv("REDIRECT_URI")
	FRONTEND_REDIRECT_URI = os.Getenv("FRONTEND_REDIRECT_URI")
	DATABASE_ENDPOINT     = os.Getenv("DATABASE_ENDPOINT")
	DATABASE_NAME         = os.Getenv("DATABASE_NAME")
	DATABASE_SECRET_ARN   = os.Getenv("DATABASE_SECRET_ARN")
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
		{FieldNames: []string{"id"}, SortingOrder: 1, Unique: true},
	})

	jwtOptions, err := auth.CreateJwtOptions(awsConfig, context.TODO(), JWT_CREDENTIAL_ARN, 0)
	if err != nil {
		return err
	}

	httpRouter := router.NewRouter(routecontext.Context{
		JwtOptions: jwtOptions,
		Database:   databaseHandle,
	})

	httpRouter.AddRoute("GET", "/api/admin/finduser", info.HandleInfo)
	httpRouter.AddRoute("GET", "/api/admin/findproject", info.HandleInfo)

	httpRouter.AddRoute("POST", "/api/admin/updaterole", register.HandleRegister)
	httpRouter.AddRoute("POST", "/api/admin/updatesubscription", register.HandleRegister)
	httpRouter.AddRoute("POST", "/api/admin/deleteuser", register.HandleRegister)
	httpRouter.AddRoute("POST", "/api/admin/deleteproject", register.HandleRegister)

	lambda.Start(httpRouter.Route)

	return nil
}
