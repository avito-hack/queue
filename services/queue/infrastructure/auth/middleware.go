package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func Middleware(sessionService *SessionService, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/healthz" {
			c.Next()
			return
		}

		header := c.GetHeader("Authorization")

		scheme, tokenString, ok := strings.Cut(header, " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(tokenString) == "" {
			logger.Warn(
				"authorization header missing",
				"path",
				c.Request.URL.Path,
			)

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "unauthorized",
				"message": "missing authorization header",
			})

			return
		}

		tokenString = strings.TrimSpace(tokenString)

		userID, err := sessionService.Validate(c.Request.Context(), tokenString)
		if err != nil {
			logger.Warn(
				"token validation failed",
				"error",
				err,
				"path",
				c.Request.URL.Path,
			)

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "unauthorized",
				"message": fmt.Sprintf("validate token: %v", err),
			})

			return
		}

		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, UserIDKey, userID)
		ctx = context.WithValue(ctx, AuthorizationHeaderKey, header)

		c.Request = c.Request.WithContext(ctx)

		logger.Info(
			"authorization successful",
			"user_id",
			userID,
			"path",
			c.Request.URL.Path,
		)

		c.Next()
	}
}