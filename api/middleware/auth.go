package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BearerAuth validates the Authorization header against the expected token.
func BearerAuth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header != "Bearer "+token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
