package auth

import (
    "errors"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
    UserIDKey              contextKey = "user_id"
    AuthorizationHeaderKey contextKey = "authorization_header"
)

type JWTService struct {
    secret []byte
}

func NewJWTService(secret string) *JWTService {
    return &JWTService{
        secret: []byte(secret),
    }
}

func (j *JWTService) Generate(userID string, expiresAt time.Time) (string, error) {
    claims := jwt.MapClaims{
        "user_id": userID,
        "exp":     expiresAt.Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(j.secret)
}

func (j *JWTService) Validate(tokenString string) (string, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return j.secret, nil
    })
    if err != nil {
        return "", err
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || !token.Valid {
        return "", errors.New("invalid token")
    }

    userID, ok := claims["user_id"].(string)
    if !ok || userID == "" {
        return "", errors.New("user_id missing")
    }

    return userID, nil
}
