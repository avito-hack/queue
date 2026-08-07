package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Middleware_SkipAuthForHealthz(t *testing.T) {
	// given
	gin.SetMode(gin.TestMode)
	service := NewJWTService("test-secret")

	router := gin.New()
	router.Use(Middleware(service))
	router.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func Test_Middleware_ReturnUnauthorizedForMissingHeader(t *testing.T) {
	// given
	gin.SetMode(gin.TestMode)
	service := NewJWTService("test-secret")

	router := gin.New()
	router.Use(Middleware(service))
	router.POST("/secure", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/secure", nil)
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["code"])
}

func Test_Middleware_SetUserIDInGinAndRequestContext(t *testing.T) {
	// given
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	userID := uuid.New()
	service := NewJWTService(secret)

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{"user_id": userID.String()},
	)
	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	var gotUserIDFromGin uuid.UUID
	var gotUserIDFromRequestContext uuid.UUID

	router := gin.New()
	router.Use(Middleware(service))
	router.POST("/secure", func(c *gin.Context) {
		value, ok := c.Get(string(UserIDKey))
		require.True(t, ok)

		id, ok := value.(uuid.UUID)
		require.True(t, ok)
		gotUserIDFromGin = id

		ctxUserID, ok := UserID(c.Request.Context())
		require.True(t, ok)
		gotUserIDFromRequestContext = ctxUserID

		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/secure", nil)
	request.Header.Set("Authorization", "Bearer "+tokenString)
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, userID, gotUserIDFromGin)
	assert.Equal(t, userID, gotUserIDFromRequestContext)
}
