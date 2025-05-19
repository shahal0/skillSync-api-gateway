package routes

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var jobServiceURL = getEnv("JOB_SERVICE_URL", "http://localhost:8002")

func SetupJobRoutes(r *gin.Engine) {
	jobs := r.Group("/jobs")
	{
		jobs.POST("/post", forwardToJobService("/jobs/post"))
		jobs.GET("/", forwardToJobService("/jobs/"))
		jobs.POST("/apply", forwardToJobService("/jobs/apply"))
		
	}
}

func forwardToJobService(path string) gin.HandlerFunc {
	client := &http.Client{Timeout: 15 * time.Second}
	return func(c *gin.Context) {
		targetURL := jobServiceURL + path
		log.Printf("Forwarding request to: %s", targetURL)

		req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			log.Printf("Error creating request for %s: %v", targetURL, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create forward request"})
			return
		}
		req.Header = c.Request.Header.Clone()
		if clientIP := c.ClientIP(); clientIP != "" {
			req.Header.Set("X-Forwarded-For", clientIP)
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error calling job service at %s: %v", targetURL, err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to call job service"})
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body from %s: %v", targetURL, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from job service"})
			return
		}
		for key, values := range resp.Header {
			for _, value := range values {
				c.Writer.Header().Add(key, value)
			}
		}
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Printf("Environment variable %s not set, using default: %s", key, fallback)
	return fallback
}