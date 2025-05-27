package utils

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)
func ExtractToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", http.ErrNoCookie
	}

	// Ensure the token is in the correct format (Bearer token)
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", http.ErrNotSupported
	}

	// Extract the token by removing the "Bearer " prefix
	token := strings.TrimPrefix(authHeader, "Bearer ")
	return token, nil
}