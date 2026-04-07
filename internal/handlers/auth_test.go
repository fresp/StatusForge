package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/fresp/Statora/internal/models"
	authservice "github.com/fresp/Statora/internal/services/auth"
)

type stubLoginService struct {
	result *authservice.LoginResult
	err    error
}

type stubMeUserRepo struct {
	user *models.User
	err  error
}

func (s *stubLoginService) Login(_ context.Context, _ authservice.LoginRequest) (*authservice.LoginResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

func (r *stubMeUserRepo) FindByID(_ context.Context, _ string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}

func TestLoginHandlerReturnsExtendedContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	result := &authservice.LoginResult{
		Token:       "token-123",
		MFARequired: true,
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
	assert.Equal(t, true, response["mfaRequired"])

	userResp, ok := response["user"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "admin", userResp["role"])
}

func TestGetMeReturnsMFAFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := &models.User{
		ID:         primitive.NewObjectID(),
		Username:   "operator-user",
		Email:      "operator@example.com",
		Role:       "operator",
		MFAEnabled: true,
	}

	router := gin.New()
	router.GET("/api/auth/me", func(c *gin.Context) {
		c.Set("userId", "user-1")
		c.Set("username", "operator-user")
		c.Set("role", "operator")
		c.Set("mfaVerified", false)
		getMeWithRepo(&stubMeUserRepo{user: user})(c)
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
	assert.Equal(t, "operator@example.com", response["email"])
	assert.Equal(t, "operator", response["role"])
	assert.Equal(t, true, response["mfaEnabled"])
	assert.Equal(t, false, response["mfaVerified"])
}

func TestGetMeReturnsUnauthorizedWhenUserLookupFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/auth/me", func(c *gin.Context) {
		c.Set("userId", "missing-user")
		c.Set("username", "operator-user")
		c.Set("role", "operator")
		c.Set("mfaVerified", false)
		getMeWithRepo(&stubMeUserRepo{err: errors.New("not found")})(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
