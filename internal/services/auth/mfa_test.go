package auth

import (
	"context"
	"testing"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockMfaRepository struct {
	mock.Mock
}

func (m *MockMfaRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockMfaRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockMfaRepository) UpdateMFA(ctx context.Context, userID primitive.ObjectID, mfaEnabled bool, mfaSecret string) error {
	args := m.Called(ctx, userID, mfaEnabled, mfaSecret)
	return args.Error(0)
}

func (m *MockMfaRepository) FindUserByID(ctx context.Context, userID primitive.ObjectID) (*models.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*models.User), args.Error(1)
}

func TestGenerate(t *testing.T) {
	repo := new(MockMfaRepository)
	service := NewMFAService(repo, "12345678901234567890123456789012")

	user := &models.User{
		ID:    primitive.NewObjectID(),
		Email: "test@test.com",
	}
	repo.On("FindByEmail", mock.Anything, "test@test.com").Return(user, nil)

	qrCode, secret, err := service.Generate(context.Background(), "test@test.com")

	assert.NoError(t, err)
	assert.NotEmpty(t, qrCode)
	assert.NotEmpty(t, secret)
}

func TestEnable(t *testing.T) {
	repo := new(MockMfaRepository)
	service := NewMFAService(repo, "12345678901234567890123456789012")
	userID := primitive.NewObjectID()

	repo.On("UpdateMFA", mock.Anything, userID, true, mock.AnythingOfType("string")).Return(nil)

	err := service.Enable(context.Background(), userID, "test-secret")
	assert.NoError(t, err)
}

func TestValidate(t *testing.T) {
	encryptionKey := []byte("12345678901234567890123456789012")
	totpSecret := "ASDFASDFASDFASDF"
	encryptedSecret, err := util.Encrypt(encryptionKey, []byte(totpSecret))
	assert.NoError(t, err)

	user := &models.User{
		ID:           primitive.NewObjectID(),
		Email:        "test@test.com",
		MFAEnabled:   true,
		MFASecretEnc: encryptedSecret,
	}

	repo := new(MockMfaRepository)
	service := NewMFAService(repo, "12345678901234567890123456789012")

	repo.On("FindByEmail", mock.Anything, "test@test.com").Return(user, nil)

	passcode, err := util.GenerateTOTP(totpSecret)
	assert.NoError(t, err)

	valid, err := service.Validate(context.Background(), "test@test.com", passcode)
	assert.NoError(t, err)
	assert.True(t, valid)
}
