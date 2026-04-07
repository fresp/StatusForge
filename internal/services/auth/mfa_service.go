package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/fresp/Statora/internal/models"
	"github.com/fresp/Statora/internal/repository"
)

const (
	defaultMFAIssuer          = "Statora"
	defaultRecoveryCodesCount = 8
	totpPeriodSeconds         = 30
	totpDigits                = 6
)

var (
	ErrMFARequired          = errors.New("mfa is required")
	ErrInvalidMFACode       = errors.New("invalid mfa code")
	ErrInvalidPassword      = errors.New("invalid password")
	ErrMFASecretUnavailable = errors.New("mfa secret unavailable")
)

type MFAService struct {
	repo         repository.UserRepository
	jwtSecret    string
	mfaSecretKey string
	issuer       string
	now          func() time.Time
}

type StartEnrollmentResult struct {
	Secret        string
	OTPAuthURL    string
	RecoveryCodes []string
}

type VerifyEnrollmentRequest struct {
	UserID string
	Code   string
}

type VerifyChallengeRequest struct {
	UserID string
	Code   string
}

type VerifyChallengeResult struct {
	Token       string
	MFAVerified bool
}

type DisableMFARequest struct {
	UserID   string
	Password string
	Code     string
}

type UpdateProfileRequest struct {
	UserID          string
	Username        string
	CurrentPassword string
	NewPassword     string
}

func NewMFAService(repo repository.UserRepository, jwtSecret, mfaSecretKey, issuer string) *MFAService {
	resolvedIssuer := strings.TrimSpace(issuer)
	if resolvedIssuer == "" {
		resolvedIssuer = defaultMFAIssuer
	}

	return &MFAService{
		repo:         repo,
		jwtSecret:    jwtSecret,
		mfaSecretKey: mfaSecretKey,
		issuer:       resolvedIssuer,
		now:          time.Now,
	}
}

func (s *MFAService) StartEnrollment(ctx context.Context, userID string) (*StartEnrollmentResult, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	secret, err := generateTOTPSecret()
	if err != nil {
		return nil, err
	}

	encSecret, err := encryptMFASecret(secret, s.mfaSecretKey)
	if err != nil {
		return nil, err
	}

	recoveryCodes, recoveryHashes, err := generateRecoveryCodes(defaultRecoveryCodesCount)
	if err != nil {
		return nil, err
	}

	if err := s.repo.BeginMFAEnrollment(ctx, userID, encSecret, recoveryHashes); err != nil {
		return nil, err
	}

	result := &StartEnrollmentResult{
		Secret:        secret,
		OTPAuthURL:    buildOTPAuthURL(s.issuer, user.Email, secret),
		RecoveryCodes: recoveryCodes,
	}

	return result, nil
}

func (s *MFAService) VerifyEnrollment(ctx context.Context, req VerifyEnrollmentRequest) (*VerifyChallengeResult, error) {
	user, err := s.repo.FindByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	secret, err := decryptMFASecret(strings.TrimSpace(user.MFASecretEnc), s.mfaSecretKey)
	if err != nil {
		return nil, ErrMFASecretUnavailable
	}

	if !verifyTOTPCode(secret, req.Code, s.now()) {
		return nil, ErrInvalidMFACode
	}

	if err := s.repo.EnableMFA(ctx, req.UserID); err != nil {
		return nil, err
	}

	return s.buildVerifiedTokenResult(user)
}

func (s *MFAService) VerifyChallenge(ctx context.Context, req VerifyChallengeRequest) (*VerifyChallengeResult, error) {
	user, err := s.repo.FindByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	if !user.MFAEnabled {
		return nil, ErrMFARequired
	}

	if err := s.verifyMFAFactor(ctx, req.UserID, user.MFASecretEnc, user.MFARecoveryCodesHash, req.Code); err != nil {
		return nil, err
	}

	return s.buildVerifiedTokenResult(user)
}

func (s *MFAService) DisableMFA(ctx context.Context, req DisableMFARequest) error {
	user, err := s.repo.FindByID(ctx, req.UserID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return ErrInvalidPassword
	}

	if !user.MFAEnabled {
		return ErrMFARequired
	}

	if err := s.verifyMFAFactor(ctx, req.UserID, user.MFASecretEnc, user.MFARecoveryCodesHash, req.Code); err != nil {
		return err
	}

	return s.repo.DisableMFA(ctx, req.UserID)
}

func (s *MFAService) UpdateProfile(ctx context.Context, req UpdateProfileRequest) error {
	user, err := s.repo.FindByID(ctx, req.UserID)
	if err != nil {
		return err
	}

	var passwordHash *string
	if strings.TrimSpace(req.NewPassword) != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
			return ErrInvalidPassword
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		hashString := string(hashedPassword)
		passwordHash = &hashString
	}

	return s.repo.UpdateProfile(ctx, req.UserID, req.Username, passwordHash)
}

func (s *MFAService) verifyMFAFactor(ctx context.Context, userID, secretEnc string, recoveryHashes []string, code string) error {
	secret, err := decryptMFASecret(strings.TrimSpace(secretEnc), s.mfaSecretKey)
	if err != nil {
		return ErrMFASecretUnavailable
	}

	if verifyTOTPCode(secret, code, s.now()) {
		return nil
	}

	updatedHashes, consumed := consumeRecoveryCode(recoveryHashes, code)
	if !consumed {
		return ErrInvalidMFACode
	}

	if err := s.repo.ReplaceRecoveryCodes(ctx, userID, updatedHashes); err != nil {
		return err
	}

	return nil
}

func (s *MFAService) buildVerifiedTokenResult(user *models.User) (*VerifyChallengeResult, error) {
	role := strings.TrimSpace(user.Role)
	if role == "" {
		role = "admin"
	}

	token, err := generateAccessToken(user.ID.Hex(), user.Username, role, true, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &VerifyChallengeResult{Token: token, MFAVerified: true}, nil
}

func buildOTPAuthURL(issuer, accountName, secret string) string {
	resolvedAccount := strings.TrimSpace(accountName)
	if resolvedAccount == "" {
		resolvedAccount = "user"
	}

	label := url.PathEscape(issuer + ":" + resolvedAccount)
	query := url.Values{}
	query.Set("secret", secret)
	query.Set("issuer", issuer)
	query.Set("algorithm", "SHA1")
	query.Set("digits", strconv.Itoa(totpDigits))
	query.Set("period", strconv.Itoa(totpPeriodSeconds))

	return "otpauth://totp/" + label + "?" + query.Encode()
}

func generateTOTPSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}

	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	return encoder.EncodeToString(b), nil
}

func generateRecoveryCodes(count int) ([]string, []string, error) {
	codes := make([]string, 0, count)
	hashes := make([]string, 0, count)

	for i := 0; i < count; i++ {
		code, err := generateRecoveryCode()
		if err != nil {
			return nil, nil, err
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, err
		}

		codes = append(codes, code)
		hashes = append(hashes, string(hash))
	}

	return codes, hashes, nil
}

func generateRecoveryCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	chars := make([]byte, 10)
	for i, v := range b {
		chars[i] = alphabet[int(v)%len(alphabet)]
	}

	return string(chars[:5]) + "-" + string(chars[5:]), nil
}

func verifyTOTPCode(secret, code string, now time.Time) bool {
	trimmedCode := strings.TrimSpace(code)
	if len(trimmedCode) != totpDigits {
		return false
	}

	for offset := -1; offset <= 1; offset++ {
		checkTime := now.Add(time.Duration(offset*totpPeriodSeconds) * time.Second)
		expected, err := generateTOTPCode(secret, checkTime)
		if err != nil {
			return false
		}
		if hmac.Equal([]byte(expected), []byte(trimmedCode)) {
			return true
		}
	}

	return false
}

func generateTOTPCode(secret string, at time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", err
	}

	counter := uint64(at.Unix() / totpPeriodSeconds)
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, counter)

	mac := hmac.New(sha1.New, key)
	if _, err := mac.Write(counterBytes); err != nil {
		return "", err
	}
	hash := mac.Sum(nil)

	offset := int(hash[len(hash)-1] & 0x0f)
	binaryCode := (int(hash[offset]&0x7f) << 24) |
		(int(hash[offset+1]) << 16) |
		(int(hash[offset+2]) << 8) |
		int(hash[offset+3])

	otp := binaryCode % 1000000
	return fmt.Sprintf("%06d", otp), nil
}

func consumeRecoveryCode(recoveryHashes []string, code string) ([]string, bool) {
	updated := make([]string, 0, len(recoveryHashes))
	consumed := false

	for _, hash := range recoveryHashes {
		if !consumed && bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)) == nil {
			consumed = true
			continue
		}
		updated = append(updated, hash)
	}

	return updated, consumed
}
