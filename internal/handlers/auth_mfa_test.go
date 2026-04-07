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
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/fresp/Statora/internal/models"
	authservice "github.com/fresp/Statora/internal/services/auth"
)

func TestHandleMFAHandlerErrorReturnsBadRequestForInvalidMFACode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handleMFAHandlerError(c, authservice.ErrInvalidMFACode)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, authservice.ErrInvalidMFACode.Error(), response["error"])
}

func TestHandleMFAHandlerErrorReturnsUnauthorizedForInvalidPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handleMFAHandlerError(c, authservice.ErrInvalidPassword)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, authservice.ErrInvalidPassword.Error(), response["error"])
}

type stubMFAHandlerService struct {
	startEnrollmentResult *authservice.StartEnrollmentResult
	verifyResult          *authservice.VerifyChallengeResult
	err                   error

	startEnrollmentUserID string
	verifyEnrollmentReq   authservice.VerifyEnrollmentRequest
	verifyChallengeReq    authservice.VerifyChallengeRequest
	disableReq            authservice.DisableMFARequest
	updateProfileReq      authservice.UpdateProfileRequest
}

type stubMFAHandlerUserRepo struct {
	user *models.User
	err  error
}

func (s *stubMFAHandlerService) StartEnrollment(_ context.Context, userID string) (*authservice.StartEnrollmentResult, error) {
	s.startEnrollmentUserID = userID
	if s.err != nil {
		return nil, s.err
	}
	return s.startEnrollmentResult, nil
}

func (s *stubMFAHandlerService) VerifyEnrollment(_ context.Context, req authservice.VerifyEnrollmentRequest) (*authservice.VerifyChallengeResult, error) {
	s.verifyEnrollmentReq = req
	if s.err != nil {
		return nil, s.err
	}
	return s.verifyResult, nil
}

func (s *stubMFAHandlerService) VerifyChallenge(_ context.Context, req authservice.VerifyChallengeRequest) (*authservice.VerifyChallengeResult, error) {
	s.verifyChallengeReq = req
	if s.err != nil {
		return nil, s.err
	}
	return s.verifyResult, nil
}

func (s *stubMFAHandlerService) DisableMFA(_ context.Context, req authservice.DisableMFARequest) error {
	s.disableReq = req
	return s.err
}

func (s *stubMFAHandlerService) UpdateProfile(_ context.Context, req authservice.UpdateProfileRequest) error {
	s.updateProfileReq = req
	return s.err
}

func (r *stubMFAHandlerUserRepo) FindByID(_ context.Context, _ string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}

func TestMFASetupReturnsBootstrapPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &stubMFAHandlerService{startEnrollmentResult: &authservice.StartEnrollmentResult{
		Secret:        "SECRET123",
		OTPAuthURL:    "otpauth://totp/Statora:alice@example.com?secret=SECRET123",
		RecoveryCodes: []string{"AAAAA-BBBBB", "CCCCC-DDDDD"},
	}}

	router := gin.New()
	router.POST("/api/auth/mfa/setup", func(c *gin.Context) {
		c.Set("userId", "user-1")
		mfaSetupWithService(svc)(c)
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/mfa/setup", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-1", svc.startEnrollmentUserID)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "SECRET123", response["secret"])
	assert.Equal(t, "otpauth://totp/Statora:alice@example.com?secret=SECRET123", response["otpauthUrl"])
	assert.Len(t, response["recoveryCodes"], 2)
}

func TestMFAVerifyReturnsVerifiedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := &models.User{ID: primitive.NewObjectID(), Username: "alice", Email: "alice@example.com", Role: "admin"}
	svc := &stubMFAHandlerService{verifyResult: &authservice.VerifyChallengeResult{Token: "verified-token", MFAVerified: true}}

	router := gin.New()
	router.POST("/api/auth/mfa/verify", func(c *gin.Context) {
		c.Set("userId", user.ID.Hex())
		mfaVerifyWithService(svc, &stubMFAHandlerUserRepo{user: user})(c)
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/mfa/verify", bytes.NewBuffer([]byte(`{"code":"123456"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, user.ID.Hex(), svc.verifyEnrollmentReq.UserID)
	assert.Equal(t, "123456", svc.verifyEnrollmentReq.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "verified-token", response["token"])
	assert.Equal(t, true, response["mfaVerified"])

	userResp := response["user"].(map[string]interface{})
	assert.Equal(t, "alice", userResp["username"])
	assert.Equal(t, "alice@example.com", userResp["email"])
	assert.Equal(t, "admin", userResp["role"])
}

func TestMFAVerifyUsesChallengeForEnabledMFA(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := &models.User{ID: primitive.NewObjectID(), Username: "alice", Email: "alice@example.com", Role: "admin", MFAEnabled: true}
	svc := &stubMFAHandlerService{verifyResult: &authservice.VerifyChallengeResult{Token: "verified-token", MFAVerified: true}}

	router := gin.New()
	router.POST("/api/auth/mfa/verify", func(c *gin.Context) {
		c.Set("userId", user.ID.Hex())
		mfaVerifyWithService(svc, &stubMFAHandlerUserRepo{user: user})(c)
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/mfa/verify", bytes.NewBuffer([]byte(`{"code":"123456"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, user.ID.Hex(), svc.verifyChallengeReq.UserID)
	assert.Equal(t, "123456", svc.verifyChallengeReq.Code)
	assert.Empty(t, svc.verifyEnrollmentReq.UserID)
}

func TestMFAVerifyReturnsBadRequestForInvalidMFACode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := &models.User{ID: primitive.NewObjectID(), Username: "alice", Email: "alice@example.com", Role: "admin", MFAEnabled: true}
	svc := &stubMFAHandlerService{err: authservice.ErrInvalidMFACode}

	router := gin.New()
	router.POST("/api/auth/mfa/verify", func(c *gin.Context) {
		c.Set("userId", user.ID.Hex())
		mfaVerifyWithService(svc, &stubMFAHandlerUserRepo{user: user})(c)
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/mfa/verify", bytes.NewBuffer([]byte(`{"code":"000000"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, authservice.ErrInvalidMFACode.Error(), response["error"])
}

func TestMFARecoveryVerifyReturnsVerifiedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := &models.User{ID: primitive.NewObjectID(), Username: "alice", Email: "alice@example.com", Role: "admin"}
	svc := &stubMFAHandlerService{verifyResult: &authservice.VerifyChallengeResult{Token: "verified-token", MFAVerified: true}}

	router := gin.New()
	router.POST("/api/auth/mfa/recovery/verify", func(c *gin.Context) {
		c.Set("userId", user.ID.Hex())
		mfaRecoveryVerifyWithService(svc, &stubMFAHandlerUserRepo{user: user})(c)
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/mfa/recovery/verify", bytes.NewBuffer([]byte(`{"code":"AAAAA-BBBBB"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, user.ID.Hex(), svc.verifyChallengeReq.UserID)
	assert.Equal(t, "AAAAA-BBBBB", svc.verifyChallengeReq.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "verified-token", response["token"])
	assert.Equal(t, true, response["mfaVerified"])
}

func TestMFADisableRequiresPasswordAndCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &stubMFAHandlerService{}
	router := gin.New()
	router.POST("/api/auth/mfa/disable", func(c *gin.Context) {
		c.Set("userId", "user-1")
		mfaDisableWithService(svc)(c)
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/mfa/disable", bytes.NewBuffer([]byte(`{"password":"secret123","code":"123456"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-1", svc.disableReq.UserID)
	assert.Equal(t, "secret123", svc.disableReq.Password)
	assert.Equal(t, "123456", svc.disableReq.Code)
}

func TestProfileUpdateChangesUsernameAndPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &stubMFAHandlerService{}
	router := gin.New()
	router.PATCH("/api/auth/me", func(c *gin.Context) {
		c.Set("userId", "user-1")
		profileUpdateWithService(svc)(c)
	})

	req, _ := http.NewRequest(http.MethodPatch, "/api/auth/me", bytes.NewBuffer([]byte(`{"username":"alice-two","currentPassword":"secret123","newPassword":"next-secret"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-1", svc.updateProfileReq.UserID)
	assert.Equal(t, "alice-two", svc.updateProfileReq.Username)
	assert.Equal(t, "secret123", svc.updateProfileReq.CurrentPassword)
	assert.Equal(t, "next-secret", svc.updateProfileReq.NewPassword)
}
