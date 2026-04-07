package auth

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"github.com/fresp/Statora/internal/middleware"
	"github.com/fresp/Statora/internal/models"
)

type mfaStubUserRepo struct {
	user                  *models.User
	err                   error
	beginEnrollmentSecret string
	beginEnrollmentHashes []string
	replaceRecoveryHashes []string
	updateProfileUsername string
	updateProfilePassPtr  *string
	enableMFACalled       bool
	disableMFACalled      bool
	replaceRecoveryCalled bool
	updateProfileCalled   bool
	beginEnrollmentCalled bool
	replaceRecoveryUserID string
	beginEnrollmentUserID string
	updateProfileUserID   string
	disableMFAUserID      string
	enableMFAUserID       string
}

func (r *mfaStubUserRepo) FindByEmail(_ context.Context, _ string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}

func (r *mfaStubUserRepo) FindByID(_ context.Context, _ string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}

func (r *mfaStubUserRepo) UpdateProfile(_ context.Context, id string, username string, passwordHash *string) error {
	r.updateProfileCalled = true
	r.updateProfileUserID = id
	r.updateProfileUsername = username
	r.updateProfilePassPtr = passwordHash
	return r.err
}

func (r *mfaStubUserRepo) BeginMFAEnrollment(_ context.Context, id string, secretEnc string, recoveryHashes []string) error {
	r.beginEnrollmentCalled = true
	r.beginEnrollmentUserID = id
	r.beginEnrollmentSecret = secretEnc
	r.beginEnrollmentHashes = recoveryHashes
	return r.err
}

func (r *mfaStubUserRepo) EnableMFA(_ context.Context, id string) error {
	r.enableMFACalled = true
	r.enableMFAUserID = id
	if r.user != nil {
		r.user.MFAEnabled = true
	}
	return r.err
}

func (r *mfaStubUserRepo) DisableMFA(_ context.Context, id string) error {
	r.disableMFACalled = true
	r.disableMFAUserID = id
	if r.user != nil {
		r.user.MFAEnabled = false
		r.user.MFASecretEnc = ""
		r.user.MFARecoveryCodesHash = []string{}
	}
	return r.err
}

func (r *mfaStubUserRepo) ReplaceRecoveryCodes(_ context.Context, id string, hashes []string) error {
	r.replaceRecoveryCalled = true
	r.replaceRecoveryUserID = id
	r.replaceRecoveryHashes = hashes
	if r.user != nil {
		r.user.MFARecoveryCodesHash = append([]string(nil), hashes...)
	}
	return r.err
}

func TestStartEnrollmentReturnsSecretAndRecoveryCodes(t *testing.T) {
	userID := primitive.NewObjectID()
	repo := &mfaStubUserRepo{user: &models.User{
		ID:       userID,
		Username: "alice",
		Email:    "alice@example.com",
		Role:     "admin",
	}}

	svc := NewMFAService(repo, "jwt-secret", "12345678901234567890123456789012", "")

	result, err := svc.StartEnrollment(context.Background(), userID.Hex())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Secret)
	assert.Contains(t, result.OTPAuthURL, "otpauth://totp/")
	assert.Contains(t, result.OTPAuthURL, "issuer=Statora")
	assert.Len(t, result.RecoveryCodes, defaultRecoveryCodesCount)

	assert.True(t, repo.beginEnrollmentCalled)
	assert.NotEmpty(t, repo.beginEnrollmentSecret)
	assert.Len(t, repo.beginEnrollmentHashes, defaultRecoveryCodesCount)
}

func TestVerifyEnrollmentEnablesMFAOnValidCode(t *testing.T) {
	userID := primitive.NewObjectID()
	secret := "JBSWY3DPEHPK3PXP"
	enc, err := encryptMFASecret(secret, "12345678901234567890123456789012")
	require.NoError(t, err)

	repo := &mfaStubUserRepo{user: &models.User{
		ID:           userID,
		Username:     "alice",
		Email:        "alice@example.com",
		Role:         "admin",
		MFAEnabled:   false,
		MFASecretEnc: enc,
	}}

	svc := NewMFAService(repo, "jwt-secret", "12345678901234567890123456789012", "")
	code, err := generateTOTPCode(secret, svc.now())
	require.NoError(t, err)

	result, err := svc.VerifyEnrollment(context.Background(), VerifyEnrollmentRequest{
		UserID: userID.Hex(),
		Code:   code,
	})
	require.NoError(t, err)
	assert.True(t, repo.enableMFACalled)
	assert.NotEmpty(t, result.Token)
	assert.True(t, result.MFAVerified)
}

func TestVerifyChallengeReturnsVerifiedToken(t *testing.T) {
	userID := primitive.NewObjectID()
	secret := "JBSWY3DPEHPK3PXP"
	enc, err := encryptMFASecret(secret, "12345678901234567890123456789012")
	require.NoError(t, err)

	repo := &mfaStubUserRepo{user: &models.User{
		ID:           userID,
		Username:     "alice",
		Email:        "alice@example.com",
		Role:         "admin",
		MFAEnabled:   true,
		MFASecretEnc: enc,
	}}

	svc := NewMFAService(repo, "jwt-secret", "12345678901234567890123456789012", "")
	code, err := generateTOTPCode(secret, svc.now())
	require.NoError(t, err)

	result, err := svc.VerifyChallenge(context.Background(), VerifyChallengeRequest{
		UserID: userID.Hex(),
		Code:   code,
	})
	require.NoError(t, err)
	assert.True(t, result.MFAVerified)

	claims := &middleware.Claims{}
	_, err = jwt.ParseWithClaims(result.Token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("jwt-secret"), nil
	})
	require.NoError(t, err)
	assert.True(t, claims.MFAVerified)
	assert.Equal(t, userID.Hex(), claims.UserID)
}

func TestRecoveryCodeVerificationConsumesSingleUseCode(t *testing.T) {
	userID := primitive.NewObjectID()
	secret := "JBSWY3DPEHPK3PXP"
	enc, err := encryptMFASecret(secret, "12345678901234567890123456789012")
	require.NoError(t, err)

	codeOne := "recover-one"
	codeTwo := "recover-two"
	hashOne, err := bcrypt.GenerateFromPassword([]byte(codeOne), bcrypt.DefaultCost)
	require.NoError(t, err)
	hashTwo, err := bcrypt.GenerateFromPassword([]byte(codeTwo), bcrypt.DefaultCost)
	require.NoError(t, err)

	repo := &mfaStubUserRepo{user: &models.User{
		ID:                   userID,
		Username:             "alice",
		Email:                "alice@example.com",
		Role:                 "admin",
		MFAEnabled:           true,
		MFASecretEnc:         enc,
		MFARecoveryCodesHash: []string{string(hashOne), string(hashTwo)},
	}}

	svc := NewMFAService(repo, "jwt-secret", "12345678901234567890123456789012", "")

	_, err = svc.VerifyChallenge(context.Background(), VerifyChallengeRequest{UserID: userID.Hex(), Code: codeOne})
	require.NoError(t, err)
	assert.True(t, repo.replaceRecoveryCalled)
	assert.Len(t, repo.replaceRecoveryHashes, 1)

	_, err = svc.VerifyChallenge(context.Background(), VerifyChallengeRequest{UserID: userID.Hex(), Code: codeOne})
	assert.Error(t, err)
}

func TestDisableMFARequiresPasswordAndCode(t *testing.T) {
	userID := primitive.NewObjectID()
	secret := "JBSWY3DPEHPK3PXP"
	enc, err := encryptMFASecret(secret, "12345678901234567890123456789012")
	require.NoError(t, err)
	passHash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	repo := &mfaStubUserRepo{user: &models.User{
		ID:           userID,
		Username:     "alice",
		Email:        "alice@example.com",
		Role:         "admin",
		PasswordHash: string(passHash),
		MFAEnabled:   true,
		MFASecretEnc: enc,
	}}

	svc := NewMFAService(repo, "jwt-secret", "12345678901234567890123456789012", "")
	code, err := generateTOTPCode(secret, svc.now())
	require.NoError(t, err)

	err = svc.DisableMFA(context.Background(), DisableMFARequest{
		UserID:   userID.Hex(),
		Password: "secret123",
		Code:     code,
	})
	require.NoError(t, err)
	assert.True(t, repo.disableMFACalled)
}

func TestUpdateProfileHashesNewPasswordOnlyWhenProvided(t *testing.T) {
	userID := primitive.NewObjectID()
	passHash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	repo := &mfaStubUserRepo{user: &models.User{
		ID:           userID,
		Username:     "alice",
		Email:        "alice@example.com",
		Role:         "admin",
		PasswordHash: string(passHash),
	}}

	svc := NewMFAService(repo, "jwt-secret", "12345678901234567890123456789012", "")

	err = svc.UpdateProfile(context.Background(), UpdateProfileRequest{
		UserID:   userID.Hex(),
		Username: "alice-two",
	})
	require.NoError(t, err)
	assert.True(t, repo.updateProfileCalled)
	assert.Nil(t, repo.updateProfilePassPtr)

	err = svc.UpdateProfile(context.Background(), UpdateProfileRequest{
		UserID:          userID.Hex(),
		Username:        "alice-three",
		CurrentPassword: "secret123",
		NewPassword:     "next-secret",
	})
	require.NoError(t, err)
	require.NotNil(t, repo.updateProfilePassPtr)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(*repo.updateProfilePassPtr), []byte("next-secret")))
}
