package http

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	middleware "github.com/oapi-codegen/gin-middleware"

	"github.com/avito-hack/queue/services/tickets/gen/server"
	identityauth "github.com/avito-hack/queue/services/tickets/internal/auth"
)

const authenticatedUserIDKey = "tickets_authenticated_user_id"
const authenticationErrorKey = "tickets_authentication_error"

type authenticationError string

const (
	authenticationErrorRequired    authenticationError = "required"
	authenticationErrorInvalid     authenticationError = "invalid"
	authenticationErrorUnavailable authenticationError = "unavailable"
)

var (
	errBearerTokenRequired     = errors.New("bearer token is required")
	errBearerTokenInvalid      = errors.New("bearer token is invalid")
	errUserIdentityUnavailable = errors.New("user identity service unavailable")
)

func authenticateWith(resolver identityauth.UserTokenResolver, serviceToken string) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		return authenticateRequest(ctx, input, resolver, serviceToken)
	}
}

func authenticateRequest(
	ctx context.Context,
	input *openapi3filter.AuthenticationInput,
	resolver identityauth.UserTokenResolver,
	serviceToken string,
) error {
	ginContext := middleware.GetGinContext(ctx)
	header := input.RequestValidationInput.Request.Header.Get("Authorization")
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		setAuthenticationError(ginContext, authenticationErrorRequired)
		return errBearerTokenRequired
	}
	token = strings.TrimSpace(token)

	if input.SecuritySchemeName == "serviceAuth" {
		if serviceToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(serviceToken)) != 1 {
			setAuthenticationError(ginContext, authenticationErrorInvalid)
			return errBearerTokenInvalid
		}

		return nil
	}

	if ginContext == nil {
		return errUserIdentityUnavailable
	}
	userID, err := resolver.ResolveUserID(ginContext.Request.Context(), token)
	if errors.Is(err, identityauth.ErrInvalidToken) {
		setAuthenticationError(ginContext, authenticationErrorInvalid)
		return errBearerTokenInvalid
	}
	if err != nil {
		slog.ErrorContext(ginContext.Request.Context(), "resolve user token", "error", err)
		setAuthenticationError(ginContext, authenticationErrorUnavailable)
		return errUserIdentityUnavailable
	}
	if userID == uuid.Nil {
		slog.ErrorContext(ginContext.Request.Context(), "resolve user token", "error", "empty user id")
		setAuthenticationError(ginContext, authenticationErrorUnavailable)
		return errUserIdentityUnavailable
	}
	ginContext.Set(authenticatedUserIDKey, userID)

	return nil
}

func setAuthenticationError(ctx *gin.Context, value authenticationError) {
	if ctx != nil {
		ctx.Set(authenticationErrorKey, value)
	}
}

func userIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(authenticatedUserIDKey).(uuid.UUID)
	return userID, ok && userID != uuid.Nil
}

func validationErrorHandler(ctx *gin.Context, message string, statusCode int) {
	errorCode := "bad_request"
	if statusCode == http.StatusNotFound {
		errorCode = "not_found"
	}
	authError, hasAuthError := ctx.Get(authenticationErrorKey)
	switch authError {
	case authenticationErrorRequired:
		statusCode = http.StatusUnauthorized
		errorCode = "unauthorized"
		message = errBearerTokenRequired.Error()
	case authenticationErrorInvalid:
		statusCode = http.StatusUnauthorized
		errorCode = "unauthorized"
		message = errBearerTokenInvalid.Error()
	case authenticationErrorUnavailable:
		statusCode = http.StatusInternalServerError
		errorCode = "internal_error"
		message = "internal server error"
	default:
		if hasAuthError {
			statusCode = http.StatusInternalServerError
			errorCode = "internal_error"
			message = "internal server error"
		}
	}

	ctx.AbortWithStatusJSON(statusCode, server.ErrorWithMessage{
		Error:   errorCode,
		Message: message,
	})
}
