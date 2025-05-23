package routes

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	jobpb "github.com/shahal0/skillsync-protos/gen/jobpb"
	"google.golang.org/grpc/metadata"

	"skillsync-api-gateway/clients"
	"skillsync-api-gateway/middlewares"
)

func SetupJobRoutes(r *gin.Engine) {
	// Public job routes (no authentication required)
	publicJobs := r.Group("/jobs")
	{
		publicJobs.GET("/", GetJobs) // Public endpoint for listing jobs
	}

	// Protected job routes (authentication required)
	protectedJobs := r.Group("/jobs")
	protectedJobs.Use(middlewares.JWTMiddleware())
	{
		protectedJobs.POST("/post", PostJob)
		protectedJobs.POST("/apply", ApplyToJob)
		protectedJobs.POST("/addskills", AddJobSkills) // Add skills to a job
		protectedJobs.PUT("status", UpdateJobStatus)   // Update job status
	}
}

// PostJob handles job posting requests
func PostJob(c *gin.Context) {
	// Extract user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Parse request body
	var jobRequest struct {
		Title              string            `json:"title"`
		Description        string            `json:"description"`
		Category           string            `json:"category"`
		RequiredSkills     []*jobpb.JobSkill `json:"required_skills"`
		SalaryMin          int64             `json:"salary_min"`
		SalaryMax          int64             `json:"salary_max"`
		Location           string            `json:"location"`
		ExperienceRequired int32             `json:"experience_required"`
	}

	if err := c.ShouldBindJSON(&jobRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job data: " + err.Error()})
		return
	}

	// Create gRPC request with individual fields
	req := &jobpb.PostJobRequest{
		Title:              jobRequest.Title,
		Description:        jobRequest.Description,
		Category:           jobRequest.Category,
		RequiredSkills:     jobRequest.RequiredSkills,
		SalaryMin:          jobRequest.SalaryMin,
		SalaryMax:          jobRequest.SalaryMax,
		Location:           jobRequest.Location,
		ExperienceRequired: jobRequest.ExperienceRequired,
		EmployerId:         userID.(string),
	}

	// Create a context with metadata for user identification
	ctx := context.Background()
	// Add user ID and role as metadata to the gRPC request
	md := metadata.New(map[string]string{
		"x-user-id":   userID.(string),
		"x-user-role": "employer", // For post job, the role is always employer
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	log.Printf("Sending gRPC request to JobService with user ID: %s, role: employer", userID.(string))

	// Call gRPC service with the context containing user metadata
	resp, err := clients.JobServiceClient.PostJob(ctx, req)
	if err != nil {
		log.Printf("Error calling job service PostJob: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to post job: " + err.Error()})
		return
	}

	// Format response according to proto definition
	responseData := gin.H{
		"job_id":  resp.JobId,
		"message": resp.Message,
	}
	c.JSON(http.StatusCreated, responseData)
}
// GetJobs handles job search requests
func GetJobs(c *gin.Context) {
	// Extract query parameters (all optional)
	req := &jobpb.GetJobsRequest{
		Category: c.Query("category"),
		Keyword:  c.Query("keyword"),
		Location: c.Query("location"),
	}

	// Call job service
	resp, err := clients.JobServiceClient.GetJobs(context.Background(), req)
	if err != nil {
		log.Printf("Error calling job service GetJobs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch jobs: " + err.Error()})
		return
	}

	// Transform protobuf jobs to JSON-friendly format
	jobs := make([]map[string]interface{}, 0, len(resp.Jobs))
	for _, job := range resp.Jobs {
		// Format skills
		skills := make([]map[string]string, 0, len(job.RequiredSkills))
		for _, skill := range job.RequiredSkills {
			skills = append(skills, map[string]string{
				"skill": skill.Skill,
				"proficiency": skill.Proficiency,
			})
		}

			// Create company object
		company := map[string]interface{}{
			"location": job.Location, // Default to job location
		}

		// If employer profile exists, add its fields to company
		if job.EmployerProfile != nil {
			log.Printf("API Gateway: Using employer profile from job service for job %d", job.Id)
			company["company_name"] = job.EmployerProfile.CompanyName
			company["email"] = job.EmployerProfile.Email
			company["industry"] = job.EmployerProfile.Industry
			company["website"] = job.EmployerProfile.Website
			company["location"] = job.EmployerProfile.Location
		} else {
			log.Printf("API Gateway: No employer profile found for job %d with employer ID %s", job.Id, job.EmployerId)
		}

		// Create job object
		jobMap := map[string]interface{}{
			"id":                  job.Id,
			"employer_id":         job.EmployerId,
			"title":               job.Title,
			"description":         job.Description,
			"category":            job.Category,
			"required_skills":     skills,
			"salary_min":          job.SalaryMin,
			"salary_max":          job.SalaryMax,
			"location":            job.Location,
			"experience_required": job.ExperienceRequired,
			"status":              job.Status,
			"company":             company,
		}

		jobs = append(jobs, jobMap)
	}

	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// ApplyToJob handles job application requests
func ApplyToJob(c *gin.Context) {
	log.Println("ApplyToJob called")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	log.Println("User ID: ", userID)

	// Get job ID from query parameter
	jobID := c.Query("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}
	jobIDUint, err := strconv.ParseUint(jobID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID format"})
		return
	}

	req := &jobpb.ApplyToJobRequest{
		CandidateId: userID.(string), // The user applying for the job
		JobId:       jobIDUint,       // The job being applied to, converted to uint64
	}
	ctx := context.Background()
	// Add user ID and role as metadata to the gRPC request
	md := metadata.New(map[string]string{
		"x-user-id":   userID.(string),
		"x-user-role": "candidate", // For apply job, the role is always candidate
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	log.Printf("Sending gRPC request to JobService with user ID: %s, role: candidate", userID.(string))

	// Call gRPC service with the context containing user metadata
	resp, err := clients.JobServiceClient.ApplyToJob(ctx, req)
	if err != nil {
		log.Printf("Error calling job service ApplyToJob: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply to job: " + err.Error()})
		return
	}

	// Format application response with application_id and message as per proto definition
	responseData := gin.H{
		"application_id": resp.ApplicationId,
		"message":        resp.Message,
	}
	c.JSON(http.StatusOK, responseData)
}

// AddJobSkills handles adding skills to a job
func AddJobSkills(c *gin.Context) {
	// Extract user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Extract role from context
	role, exists := c.Get("user_role")
	if !exists || role.(string) != "employer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only employers can add skills to jobs"})
		return
	}

	// Parse request body
	var requestBody struct {
		JobID       uint64 `json:"job_id" binding:"required"`
		Skill       string `json:"skill" binding:"required"`
		Proficiency string `json:"proficiency" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// No need to convert job ID since the protobuf now uses string type
	jobID := requestBody.JobID

	// Create gRPC context with metadata
	ctx := context.Background()
	md := metadata.New(map[string]string{
		"authorization": c.GetHeader("Authorization"),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Create gRPC request with a single skill
	req := &jobpb.AddJobSkillsRequest{
		JobId:       uint64(jobID),
		Skill:       requestBody.Skill,
		Proficiency: requestBody.Proficiency,
	}

	log.Printf("Sending gRPC request to JobService to add skills to job %s by employer %s", requestBody.JobID, userID.(string))

	// Call gRPC service
	resp, err := clients.JobServiceClient.AddJobSkills(ctx, req)
	if err != nil {
		log.Printf("Error calling job service AddJobSkills: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add skills to job: " + err.Error()})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"message": resp.Message,
	})
}

// UpdateJobStatus handles updating a job's status
func UpdateJobStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	jobID := c.Query("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	status := c.Query("status")
	if status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status is required"})
		return
	}

	grpcReq := &jobpb.UpdateJobStatusRequest{
		JobId:      jobID,
		Status:     status,
		EmployerId: userID.(string),
	}

	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
		return
	}
	md := metadata.New(map[string]string{
		"authorization": c.GetHeader("Authorization"),
		"x-user-id":     userID.(string),
		"x-user-role":   userRole.(string),
	})

	ctx := metadata.NewOutgoingContext(context.Background(), md)

	res, err := clients.JobServiceClient.UpdateJobStatus(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": res.Message,
	})
}
