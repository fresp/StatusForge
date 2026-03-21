package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/fresp/StatusForge/configs"
	"github.com/fresp/StatusForge/internal/repository"
	authservice "github.com/fresp/StatusForge/internal/services/auth"
)

type mfaHandlerService interface {
	StartEnrollment(ctx context.Context, userID string) (*authservice.StartEnrollmentResult, error)
	VerifyEnrollment(ctx context.Context, req authservice.VerifyEnrollmentRequest) (*authservice.VerifyChallengeResult, error)
	VerifyChallenge(ctx context.Context, req authservice.VerifyChallengeRequest) (*authservice.VerifyChallengeResult, error)
	DisableMFA(ctx context.Context, req authservice.DisableMFARequest) error
	UpdateProfile(ctx context.Context, req authservice.UpdateProfileRequest) error
}

func MFASetup(db *mongo.Database, cfg *configs.Config) gin.HandlerFunc {
	userRepo := repository.NewMongoUserRepository(db)
	mfaSvc := authservice.NewMFAService(userRepo, cfg.JWTSecret, cfg.MFASecretKey, "")
	return mfaSetupWithService(mfaSvc)
}

func MFAVerify(db *mongo.Database, cfg *configs.Config) gin.HandlerFunc {
	userRepo := repository.NewMongoUserRepository(db)
	mfaSvc := authservice.NewMFAService(userRepo, cfg.JWTSecret, cfg.MFASecretKey, "")
	return mfaVerifyWithService(mfaSvc, userRepo)
}

func MFARecoveryVerify(db *mongo.Database, cfg *configs.Config) gin.HandlerFunc {
	userRepo := repository.NewMongoUserRepository(db)
	mfaSvc := authservice.NewMFAService(userRepo, cfg.JWTSecret, cfg.MFASecretKey, "")
	return mfaRecoveryVerifyWithService(mfaSvc, userRepo)
}

func MFADisable(db *mongo.Database, cfg *configs.Config) gin.HandlerFunc {
	userRepo := repository.NewMongoUserRepository(db)
	mfaSvc := authservice.NewMFAService(userRepo, cfg.JWTSecret, cfg.MFASecretKey, "")
	return mfaDisableWithService(mfaSvc)
}

func ProfileUpdate(db *mongo.Database, cfg *configs.Config) gin.HandlerFunc {
	userRepo := repository.NewMongoUserRepository(db)
	mfaSvc := authservice.NewMFAService(userRepo, cfg.JWTSecret, cfg.MFASecretKey, "")
	return profileUpdateWithService(mfaSvc)
}

func mfaSetupWithService(mfaSvc mfaHandlerService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		result, err := mfaSvc.StartEnrollment(ctx, userID)
		if err != nil {
			handleMFAHandlerError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"secret":        result.Secret,
			"otpauthUrl":    result.OTPAuthURL,
			"recoveryCodes": result.RecoveryCodes,
		})
	}
}

func mfaVerifyWithService(mfaSvc mfaHandlerService, userRepo meUserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
			return
		}

		var req struct {
			Code string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		user, err := userRepo.FindByID(ctx, userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		var result *authservice.VerifyChallengeResult
		if user.MFAEnabled {
			result, err = mfaSvc.VerifyChallenge(ctx, authservice.VerifyChallengeRequest{UserID: userID, Code: req.Code})
		} else {
			result, err = mfaSvc.VerifyEnrollment(ctx, authservice.VerifyEnrollmentRequest{UserID: userID, Code: req.Code})
		}
		if err != nil {
			handleMFAHandlerError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":       result.Token,
			"mfaVerified": result.MFAVerified,
			"user": gin.H{
				"id":       user.ID.Hex(),
				"username": user.Username,
				"email":    user.Email,
				"role":     user.Role,
			},
		})
	}
}

func mfaRecoveryVerifyWithService(mfaSvc mfaHandlerService, userRepo meUserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
			return
		}

		var req struct {
			Code string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		result, err := mfaSvc.VerifyChallenge(ctx, authservice.VerifyChallengeRequest{UserID: userID, Code: req.Code})
		if err != nil {
			handleMFAHandlerError(c, err)
			return
		}

		user, err := userRepo.FindByID(ctx, userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":       result.Token,
			"mfaVerified": result.MFAVerified,
			"user": gin.H{
				"id":       user.ID.Hex(),
				"username": user.Username,
				"email":    user.Email,
				"role":     user.Role,
			},
		})
	}
}

func mfaDisableWithService(mfaSvc mfaHandlerService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
			return
		}

		var req struct {
			Password string `json:"password" binding:"required"`
			Code     string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := mfaSvc.DisableMFA(ctx, authservice.DisableMFARequest{UserID: userID, Password: req.Password, Code: req.Code})
		if err != nil {
			handleMFAHandlerError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "mfa disabled"})
	}
}

func profileUpdateWithService(mfaSvc mfaHandlerService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
			return
		}

		var req struct {
			Username        string `json:"username" binding:"required"`
			CurrentPassword string `json:"currentPassword"`
			NewPassword     string `json:"newPassword"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := mfaSvc.UpdateProfile(ctx, authservice.UpdateProfileRequest{
			UserID:          userID,
			Username:        req.Username,
			CurrentPassword: req.CurrentPassword,
			NewPassword:     req.NewPassword,
		})
		if err != nil {
			handleMFAHandlerError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
	}
}

func getUserIDFromContext(c *gin.Context) (string, bool) {
	userID, exists := c.Get("userId")
	if !exists {
		return "", false
	}

	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		return "", false
	}

	return userIDStr, true
}

func handleMFAHandlerError(c *gin.Context, err error) {
	if errors.Is(err, authservice.ErrInvalidPassword) || errors.Is(err, authservice.ErrInvalidMFACode) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if errors.Is(err, authservice.ErrMFARequired) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}
