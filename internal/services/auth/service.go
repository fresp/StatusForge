package auth

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"status-platform/internal/repository"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type LoginRequest struct {
	Email    string
	Password string
}

type LoginResult struct {
	Token string
	Admin struct {
		ID       string
		Username string
		Email    string
		Role     string
	}
	MFARequired bool
}

type Service struct {
	repo      repository.AdminRepository
	jwtSecret string
}

func NewService(repo repository.AdminRepository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret}
}

func NewServiceFromDB(db *mongo.Database, jwtSecret string) *Service {
	return NewService(repository.NewMongoAdminRepository(db), jwtSecret)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResult, error) {
	admin, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	role := admin.Role
	if role == "" {
		role = "admin"
	}

	token, err := generateAccessToken(admin.ID.Hex(), admin.Username, role, !admin.MFAEnabled, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	var result LoginResult
	result.Token = token
	result.Admin.ID = admin.ID.Hex()
	result.Admin.Username = admin.Username
	result.Admin.Email = admin.Email
	result.Admin.Role = role
	result.MFARequired = false

	return &result, nil
}
