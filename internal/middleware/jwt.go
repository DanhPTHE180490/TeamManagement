package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// This must match the exact secret used in internal/auth/service.go
var jwtSecret = []byte("super-secret-capstone-key")

// RequireAuth is the middleware that checks for a valid JWT
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		// 2. Check if it's a Bearer token (Format: "Bearer <token>")
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			return
		}

		tokenString := parts[1]

		// 3. Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Ensure the signing method is what we expect
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// 4. Extract the data (claims) we packed into the token during Login
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// Attach the user ID and role to the Gin Context!
			// Now any handler that runs after this middleware can instantly know who is making the request.
			c.Set("userID", claims["user_id"])
			c.Set("userRole", claims["system_role"])
		}

		// 5. Token is valid, proceed to the actual handler
		c.Next()
	}
}
