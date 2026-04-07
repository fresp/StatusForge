package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"github.com/fresp/Statora/internal/models"
)

var allowedUserRoles = map[string]struct{}{
	"admin":    {},
	"operator": {},
}

var allowedUserStatuses = map[string]struct{}{
	"active":   {},
	"disabled": {},
	"invited":  {},
}

type userResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type userInvitationResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	IsExpired bool      `json:"isExpired"`
}

func mapUser(user models.User) userResponse {
	role := user.Role
	if role == "" {
		role = "admin"
	}

	status := user.Status
	if status == "" {
		status = "active"
	}

	return userResponse{
		ID:       user.ID.Hex(),
		Username: user.Username,
		Email:    user.Email,
		Role:     role,
		Status:   status,
	}
}

func clampPageToTotalPages(page, limit, total int) int {
	if page < 1 {
		return 1
	}

	if total <= 0 || limit <= 0 {
		return page
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		return page
	}

	if page > totalPages {
		return totalPages
	}

	return page
}

func GetUsers(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, limit, err := parsePaginationParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		filter := bson.M{}
		total64, err := db.Collection("users").CountDocuments(ctx, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		total := int(total64)
		page = clampPageToTotalPages(page, limit, total)

		opts := options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: 1}}).
			SetSkip(int64((page - 1) * limit)).
			SetLimit(int64(limit))

		cursor, err := db.Collection("users").Find(ctx, filter, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var users []models.User
		if err := cursor.All(ctx, &users); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		items := make([]userResponse, 0, len(users))
		for _, user := range users {
			items = append(items, mapUser(user))
		}

		if items == nil {
			items = []userResponse{}
		}

		writePaginatedResponse(c, items, total, page, limit)
	}
}

func PatchUser(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		var req struct {
			Role   string `json:"role"`
			Status string `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		set := bson.M{"updatedAt": time.Now()}

		if req.Role != "" {
			if _, ok := allowedUserRoles[req.Role]; !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
				return
			}
			set["role"] = req.Role
		}

		if req.Status != "" {
			if _, ok := allowedUserStatuses[req.Status]; !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
				return
			}
			set["status"] = req.Status
		}

		if len(set) == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no updatable fields provided"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var updated models.User
		err = db.Collection("users").FindOneAndUpdate(
			ctx,
			bson.M{"_id": id},
			bson.M{"$set": set},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).Decode(&updated)

		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, mapUser(updated))
	}
}

func DeleteUser(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		rawUserID, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user context"})
			return
		}

		currentUserIDHex, ok := rawUserID.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user context"})
			return
		}

		currentUserID, err := primitive.ObjectIDFromHex(currentUserIDHex)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user id"})
			return
		}

		if currentUserID == id {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete your own account"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		result, err := db.Collection("users").DeleteOne(ctx, bson.M{"_id": id})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if result.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func CreateUserInvitation(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
			Role  string `json:"role" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		email := strings.ToLower(strings.TrimSpace(req.Email))
		role := strings.ToLower(strings.TrimSpace(req.Role))
		if _, ok := allowedUserRoles[role]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := db.Collection("users").FindOne(ctx, bson.M{"email": email}).Err(); err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
			return
		}

		rawToken, err := generateInviteToken(32)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate invitation token"})
			return
		}

		hash := sha256.Sum256([]byte(rawToken))
		tokenHash := hex.EncodeToString(hash[:])

		createdBy := primitive.NilObjectID
		if v, ok := c.Get("userId"); ok {
			if s, sok := v.(string); sok {
				if objID, parseErr := primitive.ObjectIDFromHex(s); parseErr == nil {
					createdBy = objID
				}
			}
		}

		now := time.Now()
		invitation := models.UserInvitation{
			ID:        primitive.NewObjectID(),
			TokenHash: tokenHash,
			Email:     email,
			Role:      role,
			ExpiresAt: now.Add(48 * time.Hour),
			CreatedBy: createdBy,
			CreatedAt: now,
		}

		if _, err := db.Collection("user_invitations").InsertOne(ctx, invitation); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		activationLink := buildActivationLink(c, rawToken)

		c.JSON(http.StatusCreated, gin.H{
			"id":             invitation.ID.Hex(),
			"email":          invitation.Email,
			"role":           invitation.Role,
			"expiresAt":      invitation.ExpiresAt,
			"activationLink": activationLink,
			"token":          rawToken,
		})
	}
}

func GetUserInvitations(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, limit, err := parsePaginationParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		filter := bson.M{
			"acceptedAt": bson.M{"$exists": false},
			"revokedAt":  bson.M{"$exists": false},
		}

		total64, err := db.Collection("user_invitations").CountDocuments(ctx, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		total := int(total64)
		page = clampPageToTotalPages(page, limit, total)

		opts := options.Find().
			SetSort(bson.D{{Key: "createdAt", Value: -1}}).
			SetSkip(int64((page - 1) * limit)).
			SetLimit(int64(limit))

		cursor, err := db.Collection("user_invitations").Find(ctx, filter, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var invitations []models.UserInvitation
		if err := cursor.All(ctx, &invitations); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		now := time.Now()
		items := make([]userInvitationResponse, 0, len(invitations))
		for _, invitation := range invitations {
			items = append(items, userInvitationResponse{
				ID:        invitation.ID.Hex(),
				Email:     invitation.Email,
				Role:      invitation.Role,
				ExpiresAt: invitation.ExpiresAt,
				CreatedAt: invitation.CreatedAt,
				IsExpired: invitation.ExpiresAt.Before(now),
			})
		}

		if items == nil {
			items = []userInvitationResponse{}
		}

		writePaginatedResponse(c, items, total, page, limit)
	}
}

func RefreshUserInvitation(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		invitationID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var invitation models.UserInvitation
		err = db.Collection("user_invitations").FindOne(ctx, bson.M{
			"_id":        invitationID,
			"acceptedAt": bson.M{"$exists": false},
			"revokedAt":  bson.M{"$exists": false},
		}).Decode(&invitation)
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "invitation not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		rawToken, err := generateInviteToken(32)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate invitation token"})
			return
		}

		hash := sha256.Sum256([]byte(rawToken))
		tokenHash := hex.EncodeToString(hash[:])
		now := time.Now()
		expiresAt := now.Add(48 * time.Hour)

		_, err = db.Collection("user_invitations").UpdateByID(ctx, invitationID, bson.M{"$set": bson.M{
			"tokenHash": tokenHash,
			"expiresAt": expiresAt,
		}})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		activationLink := buildActivationLink(c, rawToken)
		c.JSON(http.StatusOK, gin.H{
			"id":             invitationID.Hex(),
			"email":          invitation.Email,
			"role":           invitation.Role,
			"expiresAt":      expiresAt,
			"activationLink": activationLink,
			"token":          rawToken,
		})
	}
}

func RevokeUserInvitation(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		invitationID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		now := time.Now()
		result, err := db.Collection("user_invitations").UpdateOne(ctx, bson.M{
			"_id":        invitationID,
			"acceptedAt": bson.M{"$exists": false},
			"revokedAt":  bson.M{"$exists": false},
		}, bson.M{"$set": bson.M{"revokedAt": now}})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "invitation not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func ActivateUserInvitation(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Token    string `json:"token" binding:"required"`
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required,min=8"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		token := strings.TrimSpace(req.Token)
		username := strings.TrimSpace(req.Username)
		if token == "" || username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token, username and password are required"})
			return
		}

		hash := sha256.Sum256([]byte(token))
		tokenHash := hex.EncodeToString(hash[:])

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var invitation models.UserInvitation
		err := db.Collection("user_invitations").FindOne(ctx, bson.M{
			"tokenHash":  tokenHash,
			"revokedAt":  bson.M{"$exists": false},
			"acceptedAt": bson.M{"$exists": false},
			"expiresAt":  bson.M{"$gt": time.Now()},
		}).Decode(&invitation)
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired invitation token"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := db.Collection("users").FindOne(ctx, bson.M{"email": invitation.Email}).Err(); err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
			return
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
			return
		}

		now := time.Now()
		user := models.User{
			ID:           primitive.NewObjectID(),
			Username:     username,
			Email:        invitation.Email,
			Role:         invitation.Role,
			Status:       "active",
			InvitedBy:    &invitation.CreatedBy,
			PasswordHash: string(passwordHash),
			CreatedAt:    now,
		}

		if _, err := db.Collection("users").InsertOne(ctx, user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if _, err := db.Collection("user_invitations").UpdateByID(ctx, invitation.ID, bson.M{"$set": bson.M{"acceptedAt": now}}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":       user.ID.Hex(),
			"email":    user.Email,
			"username": user.Username,
			"role":     user.Role,
		})
	}
}

func generateInviteToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func buildActivationLink(c *gin.Context, rawToken string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := c.GetHeader("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	}

	host := c.Request.Host
	if host == "" {
		host = "localhost"
	}

	return scheme + "://" + host + "/admin/activate?token=" + rawToken
}
