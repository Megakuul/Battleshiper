package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/golang-jwt/jwt/v5"
)

type JwtOptions struct {
	Secret string
	TTL    time.Duration
}

type UserClaims struct {
	Id        string `json:"id"`
	Provider  string `json:"provider"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	jwt.RegisteredClaims
}

type jwtCredentials struct {
	Secret string `json:"secret"`
}

// CreateJwtOptions fetches the jwtSecret containing "secret" from SecretsManager and constructs the auth.JwtOptions.
// The calling instance needs to have IAM access to the action "secretsmanager:GetSecretValue" on the provided jwtSecretARN.
func CreateJwtOptions(awsConfig aws.Config, transportCtx context.Context, jwtSecretARN string, ttl time.Duration) (*JwtOptions, error) {

	secretManagerClient := secretsmanager.NewFromConfig(awsConfig)

	secretRequest := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(jwtSecretARN),
	}

	secretResponse, err := secretManagerClient.GetSecretValue(transportCtx, secretRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire jwt secret: %v", err)
	}

	var jwtCredentials jwtCredentials
	if err := json.Unmarshal([]byte(*secretResponse.SecretString), &jwtCredentials); err != nil {
		return nil, fmt.Errorf("failed to decode jwt credential secret string: %v", err)
	}

	return &JwtOptions{
		Secret: jwtCredentials.Secret,
		TTL:    ttl,
	}, nil
}

// CreateJWT generates a jwt token based on the input options.
func CreateJWT(options *JwtOptions, id, provider, username, avatarUrl string) (string, error) {
	claims := &UserClaims{
		Id:        id,
		Provider:  provider,
		Username:  username,
		AvatarURL: avatarUrl,
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

// ParseJWT verifies the jwt string based on the provided options. It returns the user claims or an error if invalid.
func ParseJWT(options *JwtOptions, token string) (*UserClaims, error) {
	claims := &UserClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
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
