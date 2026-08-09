package auth

import (
    "context"
    "fmt"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

func Middleware(sessionService *SessionService) gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.URL.Path == "/healthz" {
            c.Next()
            return
        }

        header := c.GetHeader("Authorization")
        scheme, tokenString, ok := strings.Cut(header, " ")
        if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(tokenString) == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "missing authorization header"})
            return
        }
        tokenString = strings.TrimSpace(tokenString)

        userID, err := sessionService.Validate(c.Request.Context(), tokenString)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": fmt.Sprintf("validate token: %v", err)})
            return
        }

        ctx := c.Request.Context()
        ctx = context.WithValue(ctx, UserIDKey, userID)
        ctx = context.WithValue(ctx, AuthorizationHeaderKey, header)
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}
