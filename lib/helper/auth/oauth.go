package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"golang.org/x/oauth2"
)

type oauthCredentials struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// CreateOAuthOptions fetches the oauthSecret containing "client_id" and "client_secret" from SecretsManager and constructs the oauth.Config.
// The calling instance needs to have IAM access to the action "secretsmanager:GetSecretValue" on the provided oauthSecretARN.
func CreateOAuthOptions(awsConfig aws.Config, transportCtx context.Context, oauthSecretARN string, endpoint oauth2.Endpoint, redirectUri string, scopes []string) (*oauth2.Config, error) {

	secretManagerClient := secretsmanager.NewFromConfig(awsConfig)

	secretRequest := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(oauthSecretARN),
	}

	secretResponse, err := secretManagerClient.GetSecretValue(transportCtx, secretRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire oauth credentials: %v", err)
	}

	var oauthCredentials oauthCredentials
	if err := json.Unmarshal([]byte(*secretResponse.SecretString), &oauthCredentials); err != nil {
		return nil, fmt.Errorf("failed to decode oauth credential secret string: %v", err)
	}

	return &oauth2.Config{
		ClientID:     oauthCredentials.ClientId,
		ClientSecret: oauthCredentials.ClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  redirectUri,
		Scopes:       scopes,
	}, nil
}
