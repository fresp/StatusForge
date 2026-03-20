package auth

import (
	"context"
	"testing"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type mockMFARepo struct {
	mock.Mock
}

func (m *mockMFARepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockMFARepo) UpdateProfile(ctx context.Context, id string, username string, passwordHash *string) error {
	args := m.Called(ctx, id, username, passwordHash)
	return args.Error(0)
}

func (m *mockMFARepo) BeginMFAEnrollment(ctx context.Context, id string, secretEnc string, recoveryHashes []string) error {
	args := m.Called(ctx, id, secretEnc, recoveryHashes)
	return args.Error(0)
}

func (m *mockMFARepo) EnableMFA(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockMFARepo) DisableMFA(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockMFARepo) ReplaceRecoveryCodes(ctx context.Context, id string, hashes []string) error {
	args := m.Called(ctx, id, hashes)
	return args.Error(0)
}

func TestStartEnrollmentReturnsSecretAndRecoveryCodes(t *testing.T) {
	repo := new(mockMFARepo)
	service := NewMFAService(repo, "32-byte-test-key-123456789012345")
	userID := primitive.NewObjectID().Hex()
	user := &models.User{ID: primitive.NewObjectID(), Email: "test@test.com"}

	repo.On("FindByID", mock.Anything, userID).Return(user, nil)
	repo.On("BeginMFAEnrollment", mock.Anything, userID, mock.AnythingOfType("string"), mock.AnythingOfType("[]string")).Return(nil)

	_, _, recoveryCodes, err := service.StartEnrollment(context.Background(), userID)
	assert.NoError(t, err)
	assert.Len(t, recoveryCodes, 10)
}

func TestVerifyEnrollmentEnablesMFAOnValidCode(t *testing.T) {
	// Implementation needed
}

func TestVerifyChallengeReturnsVerifiedToken(t *testing.T) {
	// Implementation needed
}

func TestRecoveryCodeVerificationConsumesSingleUseCode(t *testing.T) {
	// Implementation needed
}

func TestDisableMFARequiresPasswordAndCode(t *testing.T) {
	// Implementation needed
}

func TestUpdateProfileHashesNewPasswordOnlyWhenProvided(t *testing.T) {
	// Implementation needed
}
