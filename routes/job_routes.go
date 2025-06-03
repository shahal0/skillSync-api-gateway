package routes

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	jobpb "github.com/shahal0/skillsync-protos/gen/jobpb"
	"google.golang.org/grpc/metadata"

	"skillsync-api-gateway/clients"
	"skillsync-api-gateway/middlewares"
)

func SetupJobRoutes(r *gin.Engine) {
	
	publicJobs := r.Group("/jobs")
	{
		publicJobs.GET("/", GetJobs)       
		publicJobs.GET("/get", GetJobById) 
	}

	protectedJobs := r.Group("/jobs")
	protectedJobs.Use(middlewares.JWTMiddleware())
	{
		protectedJobs.POST("/post", PostJob)
		protectedJobs.POST("/apply", ApplyToJob)
		protectedJobs.POST("/addskills", AddJobSkills)                
		protectedJobs.PUT("/status", UpdateJobStatus)                  
		protectedJobs.GET("/applications", GetCandidateApplications)  
		protectedJobs.GET("/application", GetApplication)              
		protectedJobs.GET("/filter-applications", FilterApplications)
		protectedJobs.GET("/applications-by-job", GetApplicationsByJob) 
	}
}

func PostJob(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	var req jobpb.PostJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.EmployerId = userID.(string)
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    "employer",
		}),
	)
	resp, err := clients.JobServiceClient.PostJob(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func GetJobs(c *gin.Context) {
	var req jobpb.GetJobsRequest
	
	// Handle query parameters directly
	if c.Query("category") != "" {
		req.Category = c.Query("category")
	}
	if c.Query("keyword") != "" {
		req.Keyword = c.Query("keyword")
	}
	if c.Query("location") != "" {
		req.Location = c.Query("location")
	}
	
	resp, err := clients.JobServiceClient.GetJobs(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func ApplyToJob(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	var req jobpb.ApplyToJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.CandidateId = userID.(string)
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    "candidate",
		}),
	)
	resp, err := clients.JobServiceClient.ApplyToJob(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply to job: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func AddJobSkills(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	var req jobpb.AddJobSkillsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    "employer",
		}),
	)
	resp, err := clients.JobServiceClient.AddJobSkills(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add skills to job: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func UpdateJobStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
		return
	}
	
	var req jobpb.UpdateJobStatusRequest
	
	// Handle query parameters directly
	req.JobId = c.Query("job_id")
	req.Status = c.Query("status")
	
	req.EmployerId = userID.(string)
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    userRole.(string),
		}),
	)
	resp, err := clients.JobServiceClient.UpdateJobStatus(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func GetJobById(c *gin.Context) {
	var req jobpb.GetJobByIdRequest
	
	// Handle query parameters directly
	jobIDStr := c.Query("id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}
	req.JobId = jobID
	resp, err := clients.JobServiceClient.GetJobById(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func GetCandidateApplications(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
		return
	}
	if userRole.(string) != "candidate" && userRole.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only candidates can view their applications"})
		return
	}
	var req jobpb.GetApplicationsRequest
	
	// Handle query parameters directly
	if c.Query("status") != "" {
		req.Status = c.Query("status")
	}
	req.CandidateId = userID.(string)
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    userRole.(string),
		}),
	)
	resp, err := clients.JobServiceClient.GetApplications(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get applications: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func GetApplicationsByJob(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != "employer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only employers can view applications for a job"})
		return
	}
	var req jobpb.GetApplicationsRequest
	
	// Handle query parameters directly
	jobIDStr := c.Query("job_id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil || jobID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}
	req.JobId = jobID
	
	if c.Query("status") != "" {
		req.Status = c.Query("status")
	}
	// EmployerId field doesn't exist in GetApplicationsRequest
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    userRole.(string),
		}),
	)
	resp, err := clients.JobServiceClient.GetApplications(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch applications: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func GetApplication(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
		return
	}
	
	var req jobpb.GetApplicationRequest
	
	// Handle query parameters directly
	applicationIDStr := c.Query("id")
	applicationID, err := strconv.ParseUint(applicationIDStr, 10, 64)
	if err != nil || applicationID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID"})
		return
	}
	req.ApplicationId = applicationID
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    userRole.(string),
		}),
	)

	// Call gRPC service to get the specific application
	resp, err := clients.JobServiceClient.GetApplication(ctx, &req)
	if err != nil {
		// Forward error from job service
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get application: " + err.Error()})
		return
	}

	// Check if application was found
	if resp.Application == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	
	c.JSON(http.StatusOK, resp)

	// Response already sent above
}

func FilterApplications(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
		return
	}

	var req jobpb.FilterApplicationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.EmployerId = userID.(string)

	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    userRole.(string),
		}),
	)

	
	resp, err := clients.JobServiceClient.FilterApplications(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to filter applications: " + err.Error()})
		return
	}

	
	c.JSON(http.StatusOK, resp)
}
