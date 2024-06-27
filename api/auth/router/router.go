package router

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

// RouteContext provides data to route handlers.
type RouteContext struct {
	CognitoClient       *cognitoidentityprovider.Client
	CognitoDomain       string
	ClientID            string
	ClientSecret        string
	RedirectURI         string
	FrontendRedirectURI string
}

// Router is a simple router that provides a lambda handler function,
// that is then routed to the corresponding route based on a RouteKey.
type Router struct {
	defaultContext RouteContext
	routes         map[string]func(events.APIGatewayV2HTTPRequest, context.Context, RouteContext) (events.APIGatewayV2HTTPResponse, error)
}

// NewRouter creates a new router for this endpoint.
// Provide a defaultContext which is provided to the route handlers.
func NewRouter(defaultContext RouteContext) *Router {
	return &Router{
		defaultContext: defaultContext,
		routes:         map[string]func(events.APIGatewayV2HTTPRequest, context.Context, RouteContext) (events.APIGatewayV2HTTPResponse, error){},
	}
}

// AddRoute adds a new handle to the router which is invoked when the routeKey matches.
// RouteKey uses the format "GET /the/fuck/out".
func (r *Router) AddRoute(routeKey string, handler func(events.APIGatewayV2HTTPRequest, context.Context, RouteContext) (events.APIGatewayV2HTTPResponse, error)) {
	r.routes[routeKey] = handler
}

// Route routes a request to the corresponding route and calls its route handler.
// If no route with matching routeKey is found, a 404 message is returned.
func (r *Router) Route(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	handler, ok := r.routes[request.RouteKey]
	if ok {
		return handler(request, ctx, r.defaultContext)
	} else {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusNotFound,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: fmt.Sprintf("No valid handler found for route key: '%s'", request.RouteKey),
		}, nil
	}
}
