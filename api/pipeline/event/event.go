package event

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"

	"github.com/megakuul/battleshiper/api/pipeline/routecontext"

	"github.com/go-playground/webhooks/v6/github"
)

var logger = log.New(os.Stderr, "PIPELINE EVENT: ", 0)

// HandleEvent receives events from github webhooks and handles them appropriately.
func HandleEvent(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	code, err := runHandleEvent(request, transportCtx, routeCtx)
	if err != nil {
		logger.Printf("warning: pipeline request rejected: %v\n", err)
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
		logger.Printf("failed to create pseudo request: %v\n", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to create pseudo request")
	}

	payload, err := routeCtx.WebhookClient.Parse(httpRequest, github.InstallationEvent, github.InstallationRepositoriesEvent, github.PushEvent)
	if err == github.ErrHMACVerificationFailed {
		return http.StatusForbidden, fmt.Errorf("failed to parse event: invalid signature")
	} else if err == github.ErrEventNotFound {
		return http.StatusNotFound, fmt.Errorf("failed to parse event: event not found")
	} else if err != nil {
		logger.Printf("failed to parse event: %v\n", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to parse event")
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
