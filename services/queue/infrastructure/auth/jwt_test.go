package auth

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_JWTService_Validate_ReturnUserID(t *testing.T) {
	// given
	secret := "test-secret"
	userID := uuid.New()
	service := NewJWTService(secret)

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{"user_id": userID.String()},
	)
	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	// when
	result, err := service.Validate(tokenString)

	// then
	require.NoError(t, err)
	assert.Equal(t, userID, result)
}

func Test_JWTService_Validate_ReturnErrorForInvalidSecret(t *testing.T) {
	// given
	service := NewJWTService("service-secret")

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{"user_id": uuid.New().String()},
	)
	tokenString, err := token.SignedString([]byte("other-secret"))
	require.NoError(t, err)

	// when
	_, err = service.Validate(tokenString)

	// then
	require.Error(t, err)
}

func Test_JWTService_Validate_ReturnErrorForMissingUserID(t *testing.T) {
	// given
	secret := "test-secret"
	service := NewJWTService(secret)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{})
	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	// when
	_, err = service.Validate(tokenString)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user_id")
}

func Test_UserID_ReturnValueFromContext(t *testing.T) {
	// given
	userID := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, userID)

	// when
	result, ok := UserID(ctx)

	// then
	require.True(t, ok)
	assert.Equal(t, userID, result)
}
