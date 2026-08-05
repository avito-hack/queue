package http

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"

	"github.com/avito-hack/queue/services/tickets/gen/server"
)

var errBearerTokenRequired = errors.New("bearer token is required")

func authenticate(_ context.Context, input *openapi3filter.AuthenticationInput) error {
	header := input.RequestValidationInput.Request.Header.Get("Authorization")
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return errBearerTokenRequired
	}

	return nil
}

func validationErrorHandler(ctx *gin.Context, message string, statusCode int) {
	errorCode := "bad_request"
	if statusCode == http.StatusNotFound {
		errorCode = "not_found"
	}
	if strings.Contains(message, errBearerTokenRequired.Error()) {
		statusCode = http.StatusUnauthorized
		errorCode = "unauthorized"
		message = errBearerTokenRequired.Error()
	}

	ctx.AbortWithStatusJSON(statusCode, server.ErrorWithMessage{
		Error:   errorCode,
		Message: message,
	})
}
