package utils

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func RespondWithError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"error": message})
}

func RespondWithSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}
