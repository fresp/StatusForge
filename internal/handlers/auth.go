package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/fresp/Statora/internal/models"
	"github.com/fresp/Statora/internal/repository"
	authservice "github.com/fresp/Statora/internal/services/auth"
)

type loginService interface {
	Login(ctx context.Context, req authservice.LoginRequest) (*authservice.LoginResult, error)
}

type meUserRepository interface {
	FindByID(ctx context.Context, id string) (*models.User, error)
}

func Login(db *mongo.Database, jwtSecret string) gin.HandlerFunc {
	authSvc := authservice.NewServiceFromDB(db, jwtSecret)
	return loginWithService(authSvc)
}

func loginWithService(authSvc loginService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		result, err := authSvc.Login(ctx, authservice.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":       result.Token,
			"mfaRequired": result.MFARequired,
			"user": gin.H{
				"id":       result.User.ID,
				"username": result.User.Username,
				"email":    result.User.Email,
				"role":     result.User.Role,
			},
		})
	}
}

func GetMe(db *mongo.Database) gin.HandlerFunc {
	return getMeWithRepo(repository.NewMongoUserRepository(db))
}

func getMeWithRepo(userRepo meUserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("userId")
		username, _ := c.Get("username")
		role, _ := c.Get("role")
		mfaVerified, _ := c.Get("mfaVerified")

		userIDStr, ok := userID.(string)
		if !ok || userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		user, err := userRepo.FindByID(ctx, userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"userId":      userID,
			"username":    username,
			"email":       user.Email,
			"role":        role,
			"mfaEnabled":  user.MFAEnabled,
			"mfaVerified": mfaVerified,
		})
	}
}
