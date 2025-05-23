package middlewares

import (
	"log"
	"net/http"
	"os"
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log the request path to help with debugging
		log.Printf("JWT Middleware: Processing request for path: %s", c.Request.URL.Path)
		
		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader == "" {
			log.Printf("JWT Middleware ERROR: Missing Authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
			return
		}
		log.Printf("JWT Middleware: Authorization header found: %s", authorizationHeader)

		// Check if the Authorization header has the Bearer prefix
		parts := strings.Split(authorizationHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Printf("JWT Middleware ERROR: Invalid Authorization format. Got: %s", authorizationHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be in format 'Bearer {token}'"})
			return
		}

		// Extract the actual token
		tokenString := parts[1]
		log.Printf("JWT Middleware: Token extracted: %s", tokenString)

		// Get JWT secret from environment variable or use fallback
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "your_jwt_secret" 
			log.Printf("JWT_SECRET environment variable not set, using fallback secret")
		}
		log.Printf("JWT Middleware: Using secret key: %s", jwtSecret)

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil {
			log.Printf("JWT Middleware ERROR: Token parsing failed: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			return
		}
		if !token.Valid {
			log.Printf("JWT Middleware ERROR: Token is invalid")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}
		log.Printf("JWT Middleware: Token validated successfully")

		// Extract user ID from token claims and set it in the context
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.Printf("JWT Middleware ERROR: Failed to extract claims from token")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Failed to extract claims from token"})
			return
		}
		log.Printf("JWT Middleware: Claims extracted: %+v", claims)

		userID, ok := claims["user_id"].(string)
		if !ok {
			log.Printf("JWT Middleware ERROR: User ID not found in token claims")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
			return
		}
		log.Printf("JWT Middleware: User ID extracted: %s", userID)

		// Set user ID in context for downstream handlers
		c.Set("user_id", userID)
		
		// Extract and set role in context if available
		if role, ok := claims["role"].(string); ok {
			c.Set("user_role", role)
			log.Printf("JWT Middleware: Role extracted and set in context: %s", role)
		}
		
		log.Printf("JWT Middleware: Authentication successful, proceeding to handler")

		c.Next()
	}
}
