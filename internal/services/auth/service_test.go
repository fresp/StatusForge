package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"github.com/fresp/StatusForge/internal/models"
)

type stubUserRepo struct {
	user *models.User
	err  error
}

func (r *stubUserRepo) FindByEmail(_ context.Context, _ string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}

func TestLoginIncludesRoleAndMFAVerifiedClaims(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	repo := &stubUserRepo{
		user: &models.User{
			ID:           primitive.NewObjectID(),
			Username:     "admin",
			Email:        "admin@example.com",
			PasswordHash: string(hash),
			Role:         "admin",
			MFAEnabled:   false,
		},
	}

	svc := NewService(repo, "test-secret")

	result, err := svc.Login(context.Background(), LoginRequest{Email: "admin@example.com", Password: "secret123"})
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Token)
	assert.Equal(t, "admin", result.User.Role)
	assert.Equal(t, false, result.MFARequired)
}

func TestLoginDefaultsRoleToAdminWhenMissing(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	repo := &stubUserRepo{
		user: &models.User{
			ID:           primitive.NewObjectID(),
			Username:     "legacy",
			Email:        "legacy@example.com",
			PasswordHash: string(hash),
			Role:         "",
			MFAEnabled:   false,
		},
	}

	svc := NewService(repo, "test-secret")

	result, err := svc.Login(context.Background(), LoginRequest{Email: "legacy@example.com", Password: "secret123"})
	assert.NoError(t, err)
	assert.Equal(t, "admin", result.User.Role)
}
