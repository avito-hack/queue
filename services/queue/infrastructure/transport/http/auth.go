package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/avito-hack/queue/services/queue/gen/server"
)

func validationErrorHandler(
	ctx *gin.Context,
	message string,
	statusCode int,
) {
	errorCode := "bad_request"

	if statusCode == http.StatusNotFound {
		errorCode = "not_found"
	}

	if strings.Contains(message, "authorization") || strings.Contains(message, "token") {
		statusCode = http.StatusUnauthorized
		errorCode = "unauthorized"
	}

	ctx.AbortWithStatusJSON(
		statusCode,
		server.ErrorResponse{
			Code:    errorCode,
			Message: message,
		},
	)
}
