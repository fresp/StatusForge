package auth

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"github.com/fresp/StatusForge/internal/repository"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type LoginRequest struct {
	Email    string
	Password string
}

type LoginResult struct {
	Token string
	User  struct {
		ID       string
		Username string
		Email    string
		Role     string
	}
	MFARequired bool
}

type Service struct {
	repo      repository.UserRepository
	jwtSecret string
}

func NewService(repo repository.UserRepository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret}
}

func NewServiceFromDB(db *mongo.Database, jwtSecret string) *Service {
	return NewService(repository.NewMongoUserRepository(db), jwtSecret)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResult, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	role := user.Role
	if role == "" {
		role = "admin"
	}

	token, err := generateAccessToken(user.ID.Hex(), user.Username, role, !user.MFAEnabled, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	var result LoginResult
	result.Token = token
	result.User.ID = user.ID.Hex()
	result.User.Username = user.Username
	result.User.Email = user.Email
	result.User.Role = role
	result.MFARequired = false

	return &result, nil
}
