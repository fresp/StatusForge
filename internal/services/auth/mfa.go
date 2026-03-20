package auth

import (
	"context"
	"fmt"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/util"
	"github.com/pquerna/otp/totp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MFAService struct {
	repo      mfaRepository
	secretKey string
}

func NewMFAService(repo mfaRepository, secretKey string) *MFAService {
	return &MFAService{repo: repo, secretKey: secretKey}
}

func (s *MFAService) Generate(ctx context.Context, email string) (string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "StatusForge",
		AccountName: email,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	return key.URL(), key.Secret(), nil
}

func (s *MFAService) Enable(ctx context.Context, userID primitive.ObjectID, secret string) error {
	encryptedSecret, err := util.Encrypt([]byte(s.secretKey), []byte(secret))
	if err != nil {
		return fmt.Errorf("failed to encrypt MFA secret: %w", err)
	}
	return s.repo.UpdateMFA(ctx, userID, true, encryptedSecret)
}

func (s *MFAService) Validate(ctx context.Context, email, passcode string) (bool, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return false, fmt.Errorf("failed to find user: %w", err)
	}

	if !user.MFAEnabled {
		return false, nil
	}

	secret, err := util.Decrypt([]byte(s.secretKey), user.MFASecretEnc)
	if err != nil {
		return false, fmt.Errorf("failed to decrypt MFA secret: %w", err)
	}

	return totp.Validate(passcode, secret), nil
}

type mfaRepository interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateMFA(ctx context.Context, userID primitive.ObjectID, mfaEnabled bool, mfaSecret string) error
}
