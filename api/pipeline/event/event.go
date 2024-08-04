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

func createPseudoRequest(body []byte, contentType string) (*http.Request, error) {
	req, err := http.NewRequest("POST", "/", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	return req, nil
}

func runHandleEvent(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (int, error) {
	hook, _ := github.New()

	httpRequest, err := createPseudoRequest([]byte(request.Body), request.Headers["content-type"])
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create pseudo request")
	}

	payload, err := hook.Parse(httpRequest)
	if err == github.ErrEventNotFound {
		return http.StatusNotFound, fmt.Errorf("failed to parse event: event not found")
	} else if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse event")
	}

	switch payload.(type) {
	case github.InstallationPayload:

	case github.InstallationRepositoriesPayload:

	case github.PushPayload:

	}

	return http.StatusOK, nil
}
