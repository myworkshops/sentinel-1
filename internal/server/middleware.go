package server

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GenerateState middleware creates a random state value, stores it in a secure cookie,
// and adds it to the request context for later use in the OAuth2 flow.
func GenerateState() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate 16 random bytes
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
			return
		}
		state := base64.URLEncoding.EncodeToString(b)

		// Set state in a cookie (secure, HttpOnly)
		c.SetCookie("oauth_state", state, 300, "/", "", false, true)

		// Store state in context for handlers to retrieve
		c.Set("state", state)

		c.Next()
	}
}

// VerifyState middleware compares the state parameter from the query
// with the value stored in the cookie to protect against CSRF attacks.
func VerifyState() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookieState, err := c.Cookie("oauth_state")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing state cookie"})
			return
		}

		queryState := c.Query("state")
		if cookieState != queryState {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid state parameter"})
			return
		}

		c.Next()
	}
}

// SecureHeaders middleware adds common security headers to all responses.
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("Referrer-Policy", "no-referrer")
		c.Next()
	}
}
