package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

var (
	clientID      = os.Getenv("CLIENT_ID")
	clientSecret  = os.Getenv("CLIENT_SECRET")
	redirectURI   = os.Getenv("REDIRECT_URI")
	cognitoDomain = os.Getenv("COGNITO_DOMAIN")
	region        = os.Getenv("AWS_REGION")
)

func handler(request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	claims := request.RequestContext.Authorizer.JWT.Claims
	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintln(claims),
		StatusCode: 200,
	}, nil
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // Expiration time of the access token
}

// AcquireTokens exchanges authCode, clientId and clientSecret with Access-, ID- and Refreshtoken.
// Follows the horrible OAuth2.0 standard which Cognito complies with:
// AccessToken request: https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.3
// With ClientSecret: https://datatracker.ietf.org/doc/html/rfc6749#section-2.3.1
func AcquireTokens(authCode, clientId, clientSecret, redirectUri string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)
	data.Set("code", authCode)
	data.Set("redirect_uri", redirectUri)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth2/token", cognitoDomain), bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var tokenRes TokenResponse
	if err := json.Unmarshal(body, &tokenRes); err != nil {
		return nil, err
	}
	return &tokenRes, nil
}

func getUserInfo(accessToken, region string) (*cognitoidentityprovider.GetUserOutput, error) {
	svc := cognitoidentityprovider.New(session.New(), aws.NewConfig().WithRegion(region))
	req := &cognitoidentityprovider.GetUserInput{
		AccessToken: accessToken,
	}
	res, err := svc.GetUser(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func main() {
	lambda.Start(handler)
}
