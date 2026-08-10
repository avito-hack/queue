package http

import (
	"fmt"

	"github.com/gin-gonic/gin"
	middleware "github.com/oapi-codegen/gin-middleware"

	"github.com/avito-hack/queue/services/avito-adapter/gen/server"
)

func NewRouter(handler *Handler) (*gin.Engine, error) {
	spec, err := server.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI specification: %w", err)
	}

	spec.Servers = nil
	router := gin.New()
	router.Use(gin.Recovery(), middleware.OapiRequestValidator(spec))
	server.RegisterHandlers(router, server.NewStrictHandler(handler, nil))

	return router, nil
}
