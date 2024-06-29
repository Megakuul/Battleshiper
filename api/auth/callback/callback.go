package callback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
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
func HandleCallback(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {

	authCode := request.QueryStringParameters["code"]

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", routeCtx.ClientID)
	data.Set("client_secret", routeCtx.ClientSecret)
	data.Set("code", authCode)
	data.Set("redirect_uri", routeCtx.RedirectURI)

	req, err := http.NewRequestWithContext(transportCtx, "POST", fmt.Sprintf("%s/oauth2/token", routeCtx.CognitoDomain), bytes.NewBufferString(data.Encode()))
	if err != nil {
		log.Printf("ERROR CALLBACK: %v\n", err)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Failed to create request for authentication provider",
		}, nil
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR CALLBACK: %v\n", err)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Failed to contact authentication provider",
		}, nil
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("ERROR CALLBACK: %v\n", err)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Failed to contact authentication provider",
		}, nil
	}

	var tokenRes TokenResponse
	if err := json.Unmarshal(body, &tokenRes); err != nil {
		log.Printf("ERROR CALLBACK: %v\n", err)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Failed to read response from authentication provider",
		}, nil
	}

	accessTokenCookie := &http.Cookie{
		Name:     "access_token",
		Value:    tokenRes.AccessToken,
		HttpOnly: false,
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
			"Location":   routeCtx.FrontendRedirectURI,
			"Set-Cookie": fmt.Sprintf("%s, %s", accessTokenCookie, refreshTokenCookie),
		},
	}, nil
}
