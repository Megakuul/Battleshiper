package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/golang-jwt/jwt/v5"
)

type TicketOptions struct {
	Secret string
	Source string
	Action string
	TTL    time.Duration
}

type TicketClaims struct {
	UserID  string `json:"user_id"`
	Project string `json:"project"`
	Source  string `json:"source"`
	Action  string `json:"action"`
	jwt.RegisteredClaims
}

type ticketCredentials struct {
	Secret string `json:"secret"`
}

// CreateTicketOptions fetches the ticketSecretArn containing "secret" from SecretsManager and constructs the TicketOptions.
// The calling instance needs to have IAM access to the action "secretsmanager:GetSecretValue" on the provided ticketSecretArn.
func CreateTicketOptions(awsConfig aws.Config, transportCtx context.Context, ticketSecretARN string, action string, ttl time.Duration) (*TicketOptions, error) {
	secretManagerClient := secretsmanager.NewFromConfig(awsConfig)

	secretRequest := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(ticketSecretARN),
	}

	secretResponse, err := secretManagerClient.GetSecretValue(transportCtx, secretRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire jwt secret: %v", err)
	}

	var ticketCredentials ticketCredentials
	if err := json.Unmarshal([]byte(*secretResponse.SecretString), &ticketCredentials); err != nil {
		return nil, fmt.Errorf("failed to decode ticket credential secret string: %v", err)
	}

	return &TicketOptions{
		Secret: ticketCredentials.Secret,
		TTL:    ttl,
	}, nil
}

// CreateTicket generates a ticket based on the input options.
func CreateTicket(options *TicketOptions, userId, project string) (string, error) {
	claims := &TicketClaims{
		UserID:  userId,
		Project: project,
		Source:  options.Source,
		Action:  options.Action,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(options.TTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(options.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ParseTicket verifies the ticket based on the provided options. It returns the ticket claims or an error if invalid.
func ParseTicket(options *TicketOptions, ticket string) (*TicketClaims, error) {
	claims := &TicketClaims{}
	parsedToken, err := jwt.ParseWithClaims(ticket, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(options.Secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		return nil, err
	}

	if !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
