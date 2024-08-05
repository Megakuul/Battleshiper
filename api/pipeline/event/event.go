package event

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"github.com/megakuul/battleshiper/api/pipeline/routecontext"

	"github.com/go-playground/webhooks/v6/github"
)

// HandleEvent receives events from github webhooks and handles them appropriately.
func HandleEvent(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	code, err := runHandleEvent(request, transportCtx, routeCtx)
	if err != nil {
		// Log with Printf like a real man
		fmt.Printf("EVENT ERROR: %s", err.Error())
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: code,
	}, nil
}

// createPseudoRequest creates a pseudo http.Request that can be parsed by the webhook client.
func createPseudoRequest(body []byte, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequest("POST", "/", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req, nil
}

func runHandleEvent(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (int, error) {
	httpRequest, err := createPseudoRequest([]byte(request.Body), request.Headers)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create pseudo request")
	}

	payload, err := routeCtx.WebhookClient.Parse(httpRequest)
	if err == github.ErrHMACVerificationFailed {
		return http.StatusForbidden, fmt.Errorf("failed to parse event: invalid signature")
	} else if err == github.ErrEventNotFound {
		return http.StatusNotFound, fmt.Errorf("failed to parse event: event not found")
	} else if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse event")
	}

	status := http.StatusOK

	switch p := payload.(type) {
	case github.InstallationPayload:
		status, err = handleAppInstallation(transportCtx, routeCtx, p)
	case github.InstallationRepositoriesPayload:
		status, err = handleRepoUpdate(transportCtx, routeCtx, p)
	case github.PushPayload:
		status, err = handleRepoPush(transportCtx, routeCtx, p)
	}
	if err != nil {
		return status, err
	}

	return status, nil
}
