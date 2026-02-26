package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTOrBearer validates the Authorization header as either a JWT token or the
// legacy shared bearer token. JWT tokens set customer_id in the context;
// bearer tokens pass through as before.
func JWTOrBearer(jwtSecret, bearerToken string) gin.HandlerFunc {
	secretBytes := []byte(jwtSecret)

	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Legacy bearer token check
		if header == "Bearer "+bearerToken {
			c.Next()
			return
		}

		// Try JWT
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		if tokenStr == header {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return secretBytes, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if customerID, ok := claims["customer_id"].(string); ok {
			c.Set("customer_id", customerID)
		}
		if adminID, ok := claims["admin_id"].(string); ok {
			c.Set("admin_id", adminID)
		}

		c.Next()
	}
}
