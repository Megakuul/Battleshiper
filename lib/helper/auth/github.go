package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	webhook "github.com/go-playground/webhooks/v6/github"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v63/github"
	"golang.org/x/oauth2"
)

type GithubAppOptions struct {
	AppId     string
	AppSecret *rsa.PrivateKey
}

type githubCredentials struct {
	AppId         string `json:"app_id"`
	AppSecret     string `json:"app_secret"`
	WebhookSecret string `json:"webhook_secret"`
}

// CreateGithubAppOptions fetches the githubSecret containing "app_id" and "app_secret" from SecretsManager and creates github app options.
// The calling instance needs to have IAM access to the action "secretsmanager:GetSecretValue" on the provided githubSecret.
func CreateGithubAppOptions(awsConfig aws.Config, transportCtx context.Context, githubSecretARN string) (*GithubAppOptions, error) {
	secretManagerClient := secretsmanager.NewFromConfig(awsConfig)

	secretRequest := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(githubSecretARN),
	}

	secretResponse, err := secretManagerClient.GetSecretValue(transportCtx, secretRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire github credentials: %v", err)
	}

	var githubCredentials githubCredentials
	if err := json.Unmarshal([]byte(*secretResponse.SecretString), &githubCredentials); err != nil {
		return nil, fmt.Errorf("failed to decode github credential secret string: %v", err)
	}

	pemAppSecret := fmt.Sprintf("%s\n%s\n%s\n",
		"-----BEGIN RSA PRIVATE KEY-----",
		githubCredentials.AppSecret,
		"-----END RSA PRIVATE KEY-----",
	)
	appSecret, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(pemAppSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to parse github credential app secret")
	}

	return &GithubAppOptions{
		AppId:     githubCredentials.AppId,
		AppSecret: appSecret,
	}, nil
}

// CreateGithubAppClient creates a one time github app client from the provided app options.
// The client works for 10 minutes, afterwards the jwt expires and a new one must be created.
func CreateGithubAppClient(transportCtx context.Context, options *GithubAppOptions) (*github.Client, error) {
	claims := jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * 10).Unix(), // cannot exceed 10 minutes
		"iss": options.AppId,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(options.AppSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign github app token")
	}

	oauthClient := oauth2.NewClient(transportCtx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: signedToken,
	}))

	return github.NewClient(oauthClient), nil
}

// CreateGithubWebhookClient fetches the githubSecret containing "webhook_secret" from SecretsManager and creates a github app client.
// The calling instance needs to have IAM access to the action "secretsmanager:GetSecretValue" on the provided githubSecret.
func CreateGithubWebhookClient(awsConfig aws.Config, transportCtx context.Context, githubSecretARN string) (*webhook.Webhook, error) {
	secretManagerClient := secretsmanager.NewFromConfig(awsConfig)

	secretRequest := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(githubSecretARN),
	}

	secretResponse, err := secretManagerClient.GetSecretValue(transportCtx, secretRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire github credentials: %v", err)
	}

	var githubCredentials githubCredentials
	if err := json.Unmarshal([]byte(*secretResponse.SecretString), &githubCredentials); err != nil {
		return nil, fmt.Errorf("failed to decode github credential secret string: %v", err)
	}

	hook, err := webhook.New(webhook.Options.Secret(githubCredentials.WebhookSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook client")
	}

	return hook, nil
}
