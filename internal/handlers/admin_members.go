package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"status-platform/internal/models"
)

var allowedAdminRoles = map[string]struct{}{
	"admin":    {},
	"operator": {},
}

var allowedAdminStatuses = map[string]struct{}{
	"active":   {},
	"disabled": {},
	"invited":  {},
}

type adminMemberResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type adminInvitationResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	IsExpired bool      `json:"isExpired"`
}

func mapAdminToMember(admin models.Admin) adminMemberResponse {
	role := admin.Role
	if role == "" {
		role = "admin"
	}

	status := admin.Status
	if status == "" {
		status = "active"
	}

	return adminMemberResponse{
		ID:       admin.ID.Hex(),
		Username: admin.Username,
		Email:    admin.Email,
		Role:     role,
		Status:   status,
	}
}

func GetAdmins(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := db.Collection("admins").Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var admins []models.Admin
		if err := cursor.All(ctx, &admins); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		members := make([]adminMemberResponse, 0, len(admins))
		for _, admin := range admins {
			members = append(members, mapAdminToMember(admin))
		}

		if members == nil {
			members = []adminMemberResponse{}
		}

		c.JSON(http.StatusOK, members)
	}
}

func PatchAdmin(db *mongo.Database) gin.HandlerFunc {
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
			if _, ok := allowedAdminRoles[req.Role]; !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
				return
			}
			set["role"] = req.Role
		}

		if req.Status != "" {
			if _, ok := allowedAdminStatuses[req.Status]; !ok {
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

		var updated models.Admin
		err = db.Collection("admins").FindOneAndUpdate(
			ctx,
			bson.M{"_id": id},
			bson.M{"$set": set},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).Decode(&updated)

		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, mapAdminToMember(updated))
	}
}

func CreateAdminInvitation(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
			Role  string `json:"role" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if _, ok := allowedAdminRoles[req.Role]; !ok || req.Role == "admin" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role for invitation"})
			return
		}

		email := strings.ToLower(strings.TrimSpace(req.Email))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := db.Collection("admins").FindOne(ctx, bson.M{"email": email}).Err(); err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "admin with this email already exists"})
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
		if v, ok := c.Get("adminId"); ok {
			if s, sok := v.(string); sok {
				if objID, parseErr := primitive.ObjectIDFromHex(s); parseErr == nil {
					createdBy = objID
				}
			}
		}

		now := time.Now()
		invitation := models.AdminInvitation{
			ID:        primitive.NewObjectID(),
			TokenHash: tokenHash,
			Email:     email,
			Role:      req.Role,
			ExpiresAt: now.Add(48 * time.Hour),
			CreatedBy: createdBy,
			CreatedAt: now,
		}

		if _, err := db.Collection("admin_invitations").InsertOne(ctx, invitation); err != nil {
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

func GetAdminInvitations(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := db.Collection("admin_invitations").Find(ctx, bson.M{
			"acceptedAt": bson.M{"$exists": false},
			"revokedAt":  bson.M{"$exists": false},
		}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var invitations []models.AdminInvitation
		if err := cursor.All(ctx, &invitations); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		now := time.Now()
		items := make([]adminInvitationResponse, 0, len(invitations))
		for _, invitation := range invitations {
			items = append(items, adminInvitationResponse{
				ID:        invitation.ID.Hex(),
				Email:     invitation.Email,
				Role:      invitation.Role,
				ExpiresAt: invitation.ExpiresAt,
				CreatedAt: invitation.CreatedAt,
				IsExpired: invitation.ExpiresAt.Before(now),
			})
		}

		if items == nil {
			items = []adminInvitationResponse{}
		}

		c.JSON(http.StatusOK, items)
	}
}

func RefreshAdminInvitation(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		invitationID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var invitation models.AdminInvitation
		err = db.Collection("admin_invitations").FindOne(ctx, bson.M{
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

		_, err = db.Collection("admin_invitations").UpdateByID(ctx, invitationID, bson.M{"$set": bson.M{
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

func RevokeAdminInvitation(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		invitationID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		now := time.Now()
		result, err := db.Collection("admin_invitations").UpdateOne(ctx, bson.M{
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

func ActivateAdminInvitation(db *mongo.Database) gin.HandlerFunc {
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

		var invitation models.AdminInvitation
		err := db.Collection("admin_invitations").FindOne(ctx, bson.M{
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

		if err := db.Collection("admins").FindOne(ctx, bson.M{"email": invitation.Email}).Err(); err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "admin with this email already exists"})
			return
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
			return
		}

		now := time.Now()
		admin := models.Admin{
			ID:           primitive.NewObjectID(),
			Username:     username,
			Email:        invitation.Email,
			Role:         invitation.Role,
			Status:       "active",
			InvitedBy:    &invitation.CreatedBy,
			PasswordHash: string(passwordHash),
			CreatedAt:    now,
		}

		if _, err := db.Collection("admins").InsertOne(ctx, admin); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if _, err := db.Collection("admin_invitations").UpdateByID(ctx, invitation.ID, bson.M{"$set": bson.M{"acceptedAt": now}}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":       admin.ID.Hex(),
			"email":    admin.Email,
			"username": admin.Username,
			"role":     admin.Role,
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
