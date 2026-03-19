package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	AdminID     string `json:"adminId"`
	Username    string `json:"username"`
	Role        string `json:"role,omitempty"`
	MFAVerified bool   `json:"mfaVerified,omitempty"`
	jwt.RegisteredClaims
}

type TokenClaimsInput struct {
	AdminID     string
	Username    string
	Role        string
	MFAVerified bool
	Secret      string
	Now         time.Time
}

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		tokenStr := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("adminId", claims.AdminID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("mfaVerified", claims.MFAVerified)
		c.Next()
	}
}

func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		normalized := strings.ToLower(strings.TrimSpace(role))
		if normalized == "" {
			continue
		}
		allowed[normalized] = struct{}{}
	}

	return func(c *gin.Context) {
		rawRole, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		role, ok := rawRole.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		if _, ok := allowed[strings.ToLower(strings.TrimSpace(role))]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		c.Next()
	}
}

func GenerateToken(adminID, username, secret string) (string, error) {
	return GenerateTokenWithClaims(TokenClaimsInput{
		AdminID:     adminID,
		Username:    username,
		Role:        "",
		MFAVerified: true,
		Secret:      secret,
		Now:         time.Now(),
	})
}

func GenerateTokenWithClaims(input TokenClaimsInput) (string, error) {
	claims := &Claims{
		AdminID:     input.AdminID,
		Username:    input.Username,
		Role:        input.Role,
		MFAVerified: input.MFAVerified,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(input.Secret))
}
