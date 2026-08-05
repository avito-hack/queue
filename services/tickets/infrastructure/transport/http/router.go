package http

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	middleware "github.com/oapi-codegen/gin-middleware"

	"github.com/avito-hack/queue/services/tickets/gen/server"
)

func NewRouter(handler *Handler) (*gin.Engine, error) {
	spec, err := server.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI specification: %w", err)
	}

	spec.Servers = nil
	router := gin.New()
	router.Use(
		gin.Recovery(),
		middleware.OapiRequestValidatorWithOptions(spec, &middleware.Options{
			ErrorHandler: validationErrorHandler,
			Options: openapi3filter.Options{
				AuthenticationFunc: authenticate,
			},
		}),
	)
	server.RegisterHandlers(router, server.NewStrictHandler(handler, nil))

	return router, nil
}
