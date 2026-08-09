package http

import (
    "fmt"

    "github.com/getkin/kin-openapi/openapi3filter"
    "github.com/gin-gonic/gin"
    middleware "github.com/oapi-codegen/gin-middleware"

    "github.com/avito-hack/queue/services/queue/gen/server"
    identityauth "github.com/avito-hack/queue/services/queue/internal/auth"
)

func NewRouter(
    handler *Handler,
    resolver identityauth.UserTokenResolver,
) (*gin.Engine, error) {
    if resolver == nil {
        return nil, fmt.Errorf("create router: user token resolver is nil")
    }
    spec, err := server.GetSwagger()
    if err != nil {
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
    return router, nil
}
