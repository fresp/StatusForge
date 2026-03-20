package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequireMFAMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		mfaVerified    bool
		path           string
		expectedStatus int
	}{
		{
			name:           "should allow access to admin routes if mfa is verified",
			mfaVerified:    true,
			path:           "/admin/some-route",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "should block access to admin routes if mfa is not verified",
			mfaVerified:    false,
			path:           "/admin/some-route",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "should allow access to /auth/me if mfa is not verified",
			mfaVerified:    false,
			path:           "/auth/me",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "should allow access to /auth/mfa/validate if mfa is not verified",
			mfaVerified:    false,
			path:           "/auth/mfa/validate",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			c, r := gin.CreateTestContext(rr)

			r.Use(func(c *gin.Context) {
				c.Set("mfaVerified", tt.mfaVerified)
				c.Next()
			})
			r.Use(RequireMFA())
			r.GET(tt.path, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			c.Request, _ = http.NewRequest(http.MethodGet, tt.path, nil)
			r.ServeHTTP(c.Writer, c.Request)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
