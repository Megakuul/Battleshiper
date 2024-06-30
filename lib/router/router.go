// router package is used to route api gateway requests to multiple handlers based on the http path.
package router

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// Router is a simple router that provides a lambda handler function,
// that is then routed to the corresponding route based on a method + path.
type Router[T any] struct {
	defaultContext T
	routes         map[string]func(events.APIGatewayV2HTTPRequest, context.Context, T) (events.APIGatewayV2HTTPResponse, error)
}

// NewRouter creates a new router for this endpoint.
// Provide a defaultContext which is provided to the route handlers.
func NewRouter[T any](defaultContext T) *Router[T] {
	return &Router[T]{
		defaultContext: defaultContext,
		routes:         map[string]func(events.APIGatewayV2HTTPRequest, context.Context, T) (events.APIGatewayV2HTTPResponse, error){},
	}
}

// AddRoute adds a new handle to the router which is invoked when the method + path matches.
func (r *Router[T]) AddRoute(method string, path string, handler func(events.APIGatewayV2HTTPRequest, context.Context, T) (events.APIGatewayV2HTTPResponse, error)) {
	routeKey := fmt.Sprintf(
		"%s:%s", method, path,
	)
	r.routes[routeKey] = handler
}

// Route routes a request to the corresponding route and calls its route handler.
// If no route with matching method + path is found, a 404 message is returned.
func (r *Router[T]) Route(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	routeKey := fmt.Sprintf(
		"%s:%s", request.RequestContext.HTTP.Method, request.RequestContext.HTTP.Path,
	)

	handler, ok := r.routes[routeKey]
	if ok {
		return handler(request, ctx, r.defaultContext)
	} else {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusNotFound,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: fmt.Sprintf("No valid handler found for pattern: '%s'", routeKey),
		}, nil
	}
}
