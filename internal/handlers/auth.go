package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	authservice "github.com/fresp/StatusForge/internal/services/auth"
)

type loginService interface {
	Login(ctx context.Context, req authservice.LoginRequest) (*authservice.LoginResult, error)
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
	return func(c *gin.Context) {
		userID, _ := c.Get("userId")
		username, _ := c.Get("username")
		role, _ := c.Get("role")
		c.JSON(http.StatusOK, gin.H{
			"userId":   userID,
			"username": username,
			"role":     role,
		})
	}
}
