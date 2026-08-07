package auth

import (
    "context"
    "fmt"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

func Middleware(jwtService *JWTService) gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.URL.Path == "/healthz" {
            c.Next()
            return
        }

        header := c.GetHeader("Authorization")

        scheme, tokenString, ok := strings.Cut(header, " ")
        if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(tokenString) == "" {
            c.AbortWithStatusJSON(
                http.StatusUnauthorized,
                gin.H{
                    "code":    "unauthorized",
                    "message": "missing authorization header",
                },
            )
            return
        }

        userID, err := jwtService.Validate(tokenString)
        if err != nil {
            c.AbortWithStatusJSON(
                http.StatusUnauthorized,
                gin.H{
                    "code":    "unauthorized",
                    "message": fmt.Sprintf("validate token: %v", err),
                },
            )
            return
        }

        // gin.Context — для хендлеров
        c.Set(string(UserIDKey), userID)
        c.Set(string(AuthorizationHeaderKey), header)

        // context.Context — для usecase
        ctx := c.Request.Context()
        ctx = context.WithValue(ctx, UserIDKey, userID)
        ctx = context.WithValue(ctx, AuthorizationHeaderKey, header)
        c.Request = c.Request.WithContext(ctx)

        c.Next()
    }
}
