package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	authservice "github.com/fresp/StatusForge/internal/services/auth"
)

type stubLoginService struct {
	result *authservice.LoginResult
	err    error
}

func (s *stubLoginService) Login(_ context.Context, _ authservice.LoginRequest) (*authservice.LoginResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

func TestLoginHandlerReturnsExtendedContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	result := &authservice.LoginResult{
		Token:       "token-123",
		MFARequired: false,
	}
	result.User.ID = "abc"
	result.User.Username = "admin"
	result.User.Email = "admin@example.com"
	result.User.Role = "admin"

	router := gin.New()
	router.POST("/api/auth/login", loginWithService(&stubLoginService{result: result}))

	body, _ := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "secret123",
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "token-123", response["token"])
	assert.Equal(t, false, response["mfaRequired"])

	userResp, ok := response["user"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "admin", userResp["role"])
}

func TestGetMeReturnsRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/auth/me", func(c *gin.Context) {
		c.Set("userId", "user-1")
		c.Set("username", "operator-user")
		c.Set("role", "operator")
		GetMe(nil)(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", response["userId"])
	assert.Equal(t, "operator-user", response["username"])
	assert.Equal(t, "operator", response["role"])
}
