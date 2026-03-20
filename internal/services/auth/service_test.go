package auth

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"github.com/fresp/StatusForge/internal/middleware"
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

func (r *stubUserRepo) FindByID(_ context.Context, _ string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}

func (r *stubUserRepo) UpdateProfile(_ context.Context, _ string, _ string, _ *string) error {
	return r.err
}

func (r *stubUserRepo) BeginMFAEnrollment(_ context.Context, _ string, _ string, _ []string) error {
	return r.err
}

func (r *stubUserRepo) EnableMFA(_ context.Context, _ string) error {
	return r.err
}

func (r *stubUserRepo) DisableMFA(_ context.Context, _ string) error {
	return r.err
}

func (r *stubUserRepo) ReplaceRecoveryCodes(_ context.Context, _ string, _ []string) error {
	return r.err
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
	assert.Equal(t, true, result.MFARequired)

	parsed := &middleware.Claims{}
	_, err = jwt.ParseWithClaims(result.Token, parsed, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	assert.NoError(t, err)
	assert.False(t, parsed.MFAVerified)
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
	assert.True(t, result.MFARequired)
}

func TestLoginReturnsMFARequiredForUnenrolledUsers(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	repo := &stubUserRepo{user: &models.User{
		ID:           primitive.NewObjectID(),
		Username:     "user-one",
		Email:        "user-one@example.com",
		PasswordHash: string(hash),
		Role:         "operator",
		MFAEnabled:   false,
	}}

	svc := NewService(repo, "test-secret")
	result, err := svc.Login(context.Background(), LoginRequest{Email: "user-one@example.com", Password: "secret123"})
	assert.NoError(t, err)
	assert.True(t, result.MFARequired)
}

func TestLoginReturnsMFARequiredForEnrolledUsers(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	repo := &stubUserRepo{user: &models.User{
		ID:           primitive.NewObjectID(),
		Username:     "user-two",
		Email:        "user-two@example.com",
		PasswordHash: string(hash),
		Role:         "admin",
		MFAEnabled:   true,
	}}

	svc := NewService(repo, "test-secret")
	result, err := svc.Login(context.Background(), LoginRequest{Email: "user-two@example.com", Password: "secret123"})
	assert.NoError(t, err)
	assert.True(t, result.MFARequired)
}
