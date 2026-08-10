package http

import (
	"fmt"
	"log/slog"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	middleware "github.com/oapi-codegen/gin-middleware"

	"github.com/avito-hack/queue/services/queue/gen/server"
	identityauth "github.com/avito-hack/queue/services/queue/internal/auth"
)

func NewRouter(
	handler *Handler,
	resolver identityauth.UserTokenResolver,
	logger *slog.Logger,
) (*gin.Engine, error) {
	if resolver == nil {
		err := fmt.Errorf("create router: user token resolver is nil")

		logger.Error(
			"router creation failed",
			"error",
			err,
		)

		return nil, err
	}

	spec, err := server.GetSwagger()
	if err != nil {
		logger.Error(
			"failed to load openapi specification",
			"error",
			err,
		)

		return nil, fmt.Errorf("load OpenAPI specification: %w", err)
	}

	spec.Servers = nil

	router := gin.New()
	router.ContextWithFallback = true

	router.Use(
		gin.Recovery(),
		middleware.OapiRequestValidatorWithOptions(spec, &middleware.Options{
			ErrorHandler: validationErrorHandler,
			Options: openapi3filter.Options{
				AuthenticationFunc: authenticateWith(resolver),
			},
		}),
	)

	server.RegisterHandlers(router, server.NewStrictHandler(handler, nil))

	logger.Info(
		"router initialized",
		"routes",
		len(router.Routes()),
	)

	return router, nil
}
