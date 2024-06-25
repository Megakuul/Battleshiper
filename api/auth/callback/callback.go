package callback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // Expiration time of the access token
}

// HandleCallback is the route the user is redirected from after authorization.
// It exchanges authCode, clientId and clientSecret with Access-, ID- and Refreshtoken.
// Follows the horrible OAuth2.0 standard which Cognito complies with:
// AccessToken request spec: https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.3
// with ClientSecret: https://datatracker.ietf.org/doc/html/rfc6749#section-2.3.1
// Authorization redirect spec: https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.2
func HandleCallback(request events.APIGatewayV2HTTPRequest, providerDomain, clientId, clientSecret, redirectUri, frontendRedirect string) (events.APIGatewayV2HTTPResponse, error) {

	authCode := request.QueryStringParameters["code"]

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)
	data.Set("code", authCode)
	data.Set("redirect_uri", redirectUri)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth2/token", providerDomain), bytes.NewBufferString(data.Encode()))
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	var tokenRes TokenResponse
	if err := json.Unmarshal(body, &tokenRes); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	accessTokenCookie := &http.Cookie{
		Name:     "access_token",
		Value:    tokenRes.AccessToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Now().Add(time.Duration(tokenRes.ExpiresIn) * time.Second),
	}

	refreshTokenCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    tokenRes.RefreshToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusFound,
		Headers: map[string]string{
			"Location":   frontendRedirect,
			"Set-Cookie": fmt.Sprintf("%s, %s", accessTokenCookie, refreshTokenCookie),
		},
	}, nil
}
