package http

import (
    "context"
    "fmt"
    "log"

    "github.com/getkin/kin-openapi/openapi3filter"
    "github.com/gin-gonic/gin"
    middleware "github.com/oapi-codegen/gin-middleware"

    "github.com/avito-hack/queue/services/queue/gen/server"
    "github.com/avito-hack/queue/services/queue/infrastructure/auth"
)

func NewRouter(
    handler *Handler,
    jwtService *auth.JWTService,
) (*gin.Engine, error) {

    spec, err := server.GetSwagger()
    if err != nil {
        return nil, fmt.Errorf("load OpenAPI specification: %w", err)
    }

    spec.Servers = nil

    router := gin.New()

    router.Use(
        gin.Recovery(),
        auth.Middleware(jwtService),
        middleware.OapiRequestValidatorWithOptions(
            spec,
            &middleware.Options{
                ErrorHandler: validationErrorHandler,
                Options: openapi3filter.Options{
                    AuthenticationFunc: func(
                        _ context.Context,
                        _ *openapi3filter.AuthenticationInput,
                    ) error {
                        return nil
                    },
                },
            },
        ),
    )

    server.RegisterHandlers(
        router,
        server.NewStrictHandler(handler, nil),
    )

    log.Println("router: initialized with auth middleware before oapi validator")
    return router, nil
}
