package logout

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/user"
	"go.mongodb.org/mongo-driver/bson"
)

// HandleLogout logs the user out and revokes the used tokens.
func HandleLogout(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	cookie, code, err := runHandleLogout(request, transportCtx, routeCtx)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: code,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: err.Error(),
		}, nil
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: code,
		Headers: map[string]string{
			"Set-Cookie": cookie,
		},
	}, nil
}

func runHandleLogout(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (string, int, error) {
	clearCookieHeader := fmt.Sprintf(
		"%s, %s",
		(&http.Cookie{Name: "user_token", Expires: time.Now().Add(-24 * time.Hour)}).String(),
		(&http.Cookie{Name: "access_token", Expires: time.Now().Add(-24 * time.Hour)}).String(),
	)

	userTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("user_token")
	if err != nil {
		return "", http.StatusUnauthorized, fmt.Errorf("no user_token provided")
	}

	userToken, err := auth.ParseJWT(routeCtx.JwtOptions, userTokenCookie.Value)
	if err != nil {
		return "", http.StatusUnauthorized, fmt.Errorf("user_token is invalid: %v", err)
	}

	userCollection := routeCtx.Database.Collection(user.USER_COLLECTION)

	_, err = userCollection.UpdateOne(transportCtx, bson.M{"id": userToken}, bson.M{
		"$set": bson.M{
			"refresh_token": "",
		},
	})
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("failed to remove refresh_token")
	}

	return clearCookieHeader, http.StatusOK, nil
}
