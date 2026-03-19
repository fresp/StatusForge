package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type tokenClaims struct {
	AdminID     string `json:"adminId"`
	Username    string `json:"username"`
	Role        string `json:"role,omitempty"`
	MFAVerified bool   `json:"mfaVerified,omitempty"`
	jwt.RegisteredClaims
}

func generateAccessToken(adminID, username, role string, mfaVerified bool, secret string) (string, error) {
	claims := &tokenClaims{
		AdminID:     adminID,
		Username:    username,
		Role:        role,
		MFAVerified: mfaVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
