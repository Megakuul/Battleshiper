// database package abstracts fetching and construction of the database connection options.
package database

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type databaseCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateDatabaseOptions fetches the databaseSecret containing "username" and "password" from SecretsManager and constructs the mongo client options.
// The calling instance needs to have IAM access to the action "secretsmanager:GetSecretValue" on the provided databaseSecretARN.
func CreateDatabaseOptions(awsConfig aws.Config, databaseSecretARN, databaseEndpoint, databaseName string) (*options.ClientOptions, error) {

	secretManagerClient := secretsmanager.NewFromConfig(awsConfig)

	secretRequest := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(databaseSecretARN),
	}

	secretResponse, err := secretManagerClient.GetSecretValue(context.TODO(), secretRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire database credentials: %v", err)
	}

	var databaseCredentials databaseCredentials
	if err := json.Unmarshal([]byte(*secretResponse.SecretString), &databaseCredentials); err != nil {
		return nil, fmt.Errorf("failed to decode database credential secret string: %v", err)
	}

	databaseUri := fmt.Sprintf("mongodb://%s/%s", databaseEndpoint, databaseName)

	tlsConfig := &tls.Config{InsecureSkipVerify: false}
	databaseOptions := options.Client().ApplyURI(databaseUri).SetAuth(options.Credential{
		Username: databaseCredentials.Username,
		Password: databaseCredentials.Password,
	}).SetTLSConfig(tlsConfig)

	return databaseOptions, nil
}
