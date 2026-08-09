package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	middleware "github.com/oapi-codegen/gin-middleware"

	"github.com/avito-hack/queue/services/queue/gen/server"
	"github.com/avito-hack/queue/services/queue/internal/auth"
	a "github.com/avito-hack/queue/services/queue/infrastructure/auth"
	
)

const authenticatedUserIDKey = "queue_authenticated_user_id"
const authenticationErrorKey = "queue_authentication_error"

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

var authLogger = slog.Default()

func authenticateWith(resolver auth.UserTokenResolver) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		return authenticateRequest(ctx, input, resolver)
	}
}

func authenticateRequest(
	ctx context.Context,
	input *openapi3filter.AuthenticationInput,
	resolver auth.UserTokenResolver,
) error {
	ginContext := middleware.GetGinContext(ctx)

	if ginContext == nil {
		authLogger.Error("authentication failed: gin context missing")
		return errUserIdentityUnavailable
	}

	var header string

	if ginContext.Request != nil {
		header = ginContext.Request.Header.Get("Authorization")
	}

	if header == "" && input != nil && input.RequestValidationInput != nil && input.RequestValidationInput.Request != nil {
		header = input.RequestValidationInput.Request.Header.Get("Authorization")
	}

	scheme, token, ok := strings.Cut(header, " ")

	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		setAuthenticationError(ginContext, authenticationErrorRequired)

		authLogger.Warn(
			"authentication failed: bearer token missing",
			"path",
			ginContext.Request.URL.Path,
		)

		return errBearerTokenRequired
	}

	token = strings.TrimSpace(token)

	userID, err := resolver.ResolveUserID(ginContext.Request.Context(), token)

	if errors.Is(err, auth.ErrInvalidToken) {
		setAuthenticationError(ginContext, authenticationErrorInvalid)

		authLogger.Warn(
			"authentication failed: invalid token",
			"path",
			ginContext.Request.URL.Path,
		)

		return errBearerTokenInvalid
	}

	if err != nil {
		setAuthenticationError(ginContext, authenticationErrorUnavailable)

		authLogger.Error(
			"authentication failed: identity service unavailable",
			"error",
			err,
			"path",
			ginContext.Request.URL.Path,
		)

		ginContext.Error(err)

		return errUserIdentityUnavailable
	}

	if userID == uuid.Nil {
		setAuthenticationError(ginContext, authenticationErrorUnavailable)

		authLogger.Error(
			"authentication failed: empty user id",
			"path",
			ginContext.Request.URL.Path,
		)

		return errUserIdentityUnavailable
	}

	ginContext.Set(authenticatedUserIDKey, userID)
	
	requestContext := ginContext.Request.Context()
	
	requestContext = context.WithValue(
		requestContext,
		a.AuthorizationHeaderKey,
		header,
	)
	
	ginContext.Request = ginContext.Request.WithContext(requestContext)
	
	authLogger.Info(
		"authentication successful",
		"user_id",
		userID,
		"path",
		ginContext.Request.URL.Path,
	)

	return nil
}

func setAuthenticationError(ctx *gin.Context, value authenticationError) {
	if ctx != nil {
		ctx.Set(authenticationErrorKey, value)
	}
}

func userIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		if v, exists := ginCtx.Get(authenticatedUserIDKey); exists {
			if id, ok := v.(uuid.UUID); ok && id != uuid.Nil {
				return id, true
			}
		}
	}

	if v := ctx.Value(authenticatedUserIDKey); v != nil {
		if id, ok := v.(uuid.UUID); ok && id != uuid.Nil {
			return id, true
		}
	}

	return uuid.Nil, false
}

func validationErrorHandler(ctx *gin.Context, message string, statusCode int) {
	errorCode := "bad_request"

	if statusCode == http.StatusNotFound {
		errorCode = "not_found"
	}

	authErrorRaw, hasAuthError := ctx.Get(authenticationErrorKey)

	var authError authenticationError

	if hasAuthError {
		if v, ok := authErrorRaw.(authenticationError); ok {
			authError = v
		} else if s, ok := authErrorRaw.(string); ok {
			authError = authenticationError(s)
		}
	}

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

	authLogger.Error(
		"request validation failed",
		"status",
		statusCode,
		"code",
		errorCode,
		"message",
		message,
		"path",
		ctx.Request.URL.Path,
	)

	ctx.AbortWithStatusJSON(statusCode, server.ErrorResponse{
		Code:    errorCode,
		Message: message,
	})
}