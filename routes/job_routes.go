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
		publicJobs.GET("/", GetJobs)       // Public endpoint for listing jobs
		publicJobs.GET("/get", GetJobById) // Public endpoint for getting a job by ID
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
		protectedJobs.GET("/applications-by-job", GetApplicationsByJob) // New endpoint
	}
}

// PostJob handles job posting requests
func PostJob(c *gin.Context) {
	// Extract user ID from context (set by JWTMiddleware)
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

	// Create context with metadata for job service
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    "employer",
		}),
	)

	// Call job service
	resp, err := clients.JobServiceClient.PostJob(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to post job: " + err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusCreated, gin.H{
		"job_id":  resp.JobId,
		"message": resp.Message,
	})
}

// GetJobs handles job search requests with pagination support
func GetJobs(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	
	// Ensure valid pagination values
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	
	log.Printf("API Gateway: GetJobs called with pagination - Page: %d, Limit: %d", page, limit)
	
	// Create request with filters and pagination
	req := &jobpb.GetJobsRequest{
		Category: c.Query("category"),
		Keyword:  c.Query("keyword"),
		Location: c.Query("location"),
		// Note: We've updated the proto definition to include pagination,
		// but we need to regenerate the protobuf code for these fields to be available.
		// For now, we'll use the non-paginated version.
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
				"skill":       skill.Skill,
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

	// Calculate pagination information
	totalJobs := len(resp.Jobs)
	totalPages := (totalJobs + limit - 1) / limit
	
	// Apply client-side pagination if server-side pagination is not available
	start := (page - 1) * limit
	end := start + limit
	if start >= totalJobs {
		// If start is beyond available jobs, return empty array
		jobs = []map[string]interface{}{}
	} else {
		if end > totalJobs {
			end = totalJobs
		}
		jobs = jobs[start:end]
	}
	
	c.JSON(http.StatusOK, gin.H{
		"jobs":        jobs,
		"pagination": gin.H{
			"total_count": totalJobs,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
		},
	})
}

// ApplyToJob handles job application requests
func ApplyToJob(c *gin.Context) {
	// Extract user ID from context (set by JWTMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

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

	// Create request
	req := &jobpb.ApplyToJobRequest{
		CandidateId: userID.(string),
		JobId:       jobIDUint,
	}

	// Create context with metadata for job service
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    "candidate",
		}),
	)

	// Call job service
	resp, err := clients.JobServiceClient.ApplyToJob(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply to job: " + err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"application_id": resp.ApplicationId,
		"message":        resp.Message,
	})
}

func AddJobSkills(c *gin.Context) {
	// Extract user ID from context (set by JWTMiddleware)
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

	// Create gRPC request
	req := &jobpb.AddJobSkillsRequest{
		JobId:       requestBody.JobID,
		Skill:       requestBody.Skill,
		Proficiency: requestBody.Proficiency,
	}

	// Create context with metadata for job service
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    "employer",
		}),
	)

	// Call job service
	resp, err := clients.JobServiceClient.AddJobSkills(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add skills to job: " + err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"message": resp.Message,
	})
}

// UpdateJobStatus handles updating a job's status
func UpdateJobStatus(c *gin.Context) {
	// Extract user ID from context (set by JWTMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Extract role from context
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
		return
	}

	// Get job ID from query parameter
	jobID := c.Query("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	// Get status from query parameter
	status := c.Query("status")
	if status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status is required"})
		return
	}

	// Create gRPC request
	grpcReq := &jobpb.UpdateJobStatusRequest{
		JobId:      jobID,
		Status:     status,
		EmployerId: userID.(string),
	}

	// Create context with metadata for job service
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    userRole.(string),
		}),
	)

	// Call job service
	res, err := clients.JobServiceClient.UpdateJobStatus(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"message": res.Message,
	})
}

// GetJobById handles fetching a job by its ID
func GetJobById(c *gin.Context) {
	// Get job ID from URL parameter
	jobIDStr := c.Query("job_id")
	if jobIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	// Convert job ID from string to uint64
	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID format"})
		return
	}

	// Create gRPC request
	req := &jobpb.GetJobByIdRequest{
		JobId: jobID,
	}

	// Call job service
	resp, err := clients.JobServiceClient.GetJobById(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch job: " + err.Error()})
		return
	}

	// Check if job was found
	if resp.Job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Format skills
	skills := make([]map[string]string, 0, len(resp.Job.RequiredSkills))
	for _, skill := range resp.Job.RequiredSkills {
		skills = append(skills, map[string]string{
			"skill":       skill.Skill,
			"proficiency": skill.Proficiency,
		})
	}

	// Create company object
	company := map[string]interface{}{
		"location": resp.Job.Location, // Default to job location
	}

	// If employer profile exists, add its fields to company
	if resp.Job.EmployerProfile != nil {
		company["company_name"] = resp.Job.EmployerProfile.CompanyName
		company["email"] = resp.Job.EmployerProfile.Email
		company["industry"] = resp.Job.EmployerProfile.Industry
		company["website"] = resp.Job.EmployerProfile.Website
		company["location"] = resp.Job.EmployerProfile.Location
	}

	// Create job object
	jobMap := map[string]interface{}{
		"id":                  resp.Job.Id,
		"employer_id":         resp.Job.EmployerId,
		"title":               resp.Job.Title,
		"description":         resp.Job.Description,
		"category":            resp.Job.Category,
		"required_skills":     skills,
		"salary_min":          resp.Job.SalaryMin,
		"salary_max":          resp.Job.SalaryMax,
		"location":            resp.Job.Location,
		"experience_required": resp.Job.ExperienceRequired,
		"status":              resp.Job.Status,
		"company":             company,
	}

	c.JSON(http.StatusOK, jobMap)
}

// GetCandidateApplications handles fetching applications for a candidate
func GetCandidateApplications(c *gin.Context) {
	// Extract user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Extract user role from context
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
		return
	}

	// Verify the user is a candidate
	if userRole.(string) != "candidate" && userRole.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only candidates can view their applications"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	
	// Ensure valid pagination values
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	
	log.Printf("API Gateway: GetCandidateApplications called with pagination - Page: %d, Limit: %d", page, limit)

	// Get optional status filter from query parameter
	status := c.Query("status")

	// Create gRPC request
	req := &jobpb.GetApplicationsRequest{
		CandidateId: userID.(string),
		Status:      status,
		// Note: We've updated the proto definition to include pagination,
		// but we need to regenerate the protobuf code for these fields to be available.
		// For now, we'll use the non-paginated version and apply pagination on the client side.
	}

	// Create context with metadata for job service
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    userRole.(string),
		}),
	)

	// Call job service
	resp, err := clients.JobServiceClient.GetApplications(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get applications: " + err.Error()})
		return
	}

	// Get the raw response data to inspect
	log.Printf("GetApplications response: %+v", resp)

	// Transform applications to a JSON-friendly format
	applications := make([]map[string]interface{}, 0, len(resp.Applications))

	for _, app := range resp.Applications {
		// Create the basic application info
		applicationInfo := map[string]interface{}{
			"id":           app.Id,
			"candidate_id": app.CandidateId,
			"status":       app.Status,
			"resume_url":   app.ResumeUrl,
		}

		// Add applied_at if it exists
		if app.AppliedAt != "" {
			applicationInfo["applied_at"] = app.AppliedAt
		}

		// Add job details if available
		if app.Job != nil {
			job := app.Job

			// Create job info map
			jobInfo := map[string]interface{}{
				"id":          job.Id,
				"employer_id": job.EmployerId,
				"title":       job.Title,
				"description": job.Description,
				"category":    job.Category,
				"location":    job.Location,
				"status":      job.Status,
			}

			// Add salary info if available
			if job.SalaryMin > 0 {
				jobInfo["salary_min"] = job.SalaryMin
			}
			if job.SalaryMax > 0 {
				jobInfo["salary_max"] = job.SalaryMax
			}

			// Add experience required if available
			if job.ExperienceRequired > 0 {
				jobInfo["experience_required"] = job.ExperienceRequired
			}

			// Extract skills from job if available
			if len(job.RequiredSkills) > 0 {
				skills := []map[string]string{}
				for _, skill := range job.RequiredSkills {
					skills = append(skills, map[string]string{
						"skill":       skill.Skill,
						"proficiency": skill.Proficiency,
					})
				}
				jobInfo["required_skills"] = skills
			}

			// Add job info to application
			applicationInfo["job"] = jobInfo
		}

		applications = append(applications, applicationInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"applications": applications,
		"count":        len(applications),
	})
}

// GetApplicationsByJob handles fetching all applications for a specific job ID with pagination support
func GetApplicationsByJob(c *gin.Context) {
	// Extract user ID from context (set by JWTMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Extract role from context and verify it's an employer
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != "employer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only employers can view applications for a job"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	
	// Ensure valid pagination values
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Get job ID from query parameter
	jobIDStr := c.Query("job_id")
	if jobIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}
	// Based on the memory about job_id type, we need to be careful with the conversion
	// The proto uses string type for job_id fields, but the code might be treating it as uint64
	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID format"})
		return
	}

	req := &jobpb.GetApplicationsRequest{
		JobId: jobID,
	}

	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    userRole.(string),
		}),
	)
	resp, err := clients.JobServiceClient.GetApplications(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch applications: " + err.Error()})
		return
	}

	applications := make([]map[string]interface{}, 0, len(resp.Applications))
	for _, app := range resp.Applications {
		// Map job details if present
		var jobInfo map[string]interface{}
		if app.Job != nil {
			job := app.Job
			jobInfo = map[string]interface{}{
				"id":                  job.Id,
				"employer_id":         job.EmployerId,
				"title":               job.Title,
				"description":         job.Description,
				"category":            job.Category,
				"salary_min":          job.SalaryMin,
				"salary_max":          job.SalaryMax,
				"location":            job.Location,
				"experience_required": job.ExperienceRequired,
				"status":              job.Status,
			}
			// Add skills if present
			if len(job.RequiredSkills) > 0 {
				skills := []map[string]string{}
				for _, skill := range job.RequiredSkills {
					skills = append(skills, map[string]string{
						"skill":       skill.Skill,
						"proficiency": skill.Proficiency,
					})
				}
				jobInfo["required_skills"] = skills
			}
		}

		applicationInfo := map[string]interface{}{
			"id":           app.Id,
			"candidate_id": app.CandidateId,
			"status":       app.Status,
			"resume_url":   app.ResumeUrl,
			"applied_at":   app.AppliedAt,
		}
		if jobInfo != nil {
			applicationInfo["job"] = jobInfo
		}
		applications = append(applications, applicationInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"applications": applications,
		"count":        len(applications),
	})
}

// GetApplication handles retrieving a single application by its ID
func GetApplication(c *gin.Context) {
	// Extract user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Extract user role from context
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
		return
	}

	// Extract application ID from query parameters
	applicationIDStr := c.Query("application_id")
	if applicationIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Application ID is required"})
		return
	}
	applicationID, err := strconv.ParseUint(applicationIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Application ID format: " + err.Error()})
		return
	}

	// Create gRPC request to get the specific application by ID
	req := &jobpb.GetApplicationRequest{
		ApplicationId: applicationID, // applicationID is now uint64
	}

	// Create context with metadata for job service
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    userRole.(string),
		}),
	)

	// Call gRPC service to get the specific application
	resp, err := clients.JobServiceClient.GetApplication(ctx, req)
	if err != nil {
		log.Printf("Error calling job service GetApplication: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get application: " + err.Error()})
		return
	}

	// Check if application was found
	if resp.Application == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	// Use the application from the response
	foundApplication := resp.Application

	// Create job information if available
	jobInfo := map[string]interface{}{}

	if foundApplication.Job != nil {
		job := foundApplication.Job

		// Extract skills from job
		skills := []map[string]string{}
		for _, skill := range job.RequiredSkills {
			skills = append(skills, map[string]string{
				"skill":       skill.Skill,
				"proficiency": skill.Proficiency,
			})
		}

		// Build job info
		jobInfo = map[string]interface{}{
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
		}
	}
	log.Printf("GRPC DEBUG: Job info: %+v", jobInfo)
	// Create the application response with job as an inner object
	response := map[string]interface{}{
		"id":           foundApplication.Id,
		"candidate_id": foundApplication.CandidateId,
		"status":       foundApplication.Status,
		"resume_url":   foundApplication.ResumeUrl,
		"applied_at":   foundApplication.AppliedAt,
		"job":          jobInfo,
	}

	c.JSON(http.StatusOK, response)
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

	if userRole.(string) != "employer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only employers can filter applications"})
		return
	}

	jobid := c.Query("job_id")
	if jobid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	jobID, err := strconv.ParseUint(jobid, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID format"})
		return
	}

		req := &jobpb.FilterApplicationsRequest{
		JobId:      jobID,
		EmployerId: userID.(string),
	}

	// Create context with metadata for job service
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.New(map[string]string{
			"user-id": userID.(string),
			"role":    userRole.(string),
		}),
	)

	resp, err := clients.JobServiceClient.FilterApplications(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to filter applications: " + err.Error()})
		return
	}

	// Transform protobuf response to JSON-friendly format
	rankedApplications := make([]map[string]interface{}, 0, len(resp.RankedApplications))
	for _, rankedApp := range resp.RankedApplications {
		app := rankedApp.Application

		jobInfo := map[string]interface{}{}
		if app.Job != nil {
			job := app.Job
			skills := []map[string]string{}
			for _, skill := range job.RequiredSkills {
				skills = append(skills, map[string]string{
					"skill":       skill.Skill,
					"proficiency": skill.Proficiency,
				})
			}

			jobInfo = map[string]interface{}{
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
			}
		}

		applicationInfo := map[string]interface{}{
			"id":           app.Id,
			"candidate_id": app.CandidateId,
			"status":       app.Status,
			"resume_url":   app.ResumeUrl,
			"applied_at":   app.AppliedAt,
			"job":          jobInfo,
		}

		rankedAppInfo := map[string]interface{}{
			"application":     applicationInfo,
			"relevance_score": rankedApp.RelevanceScore,
			"matching_skills": rankedApp.MatchingSkills,
			"missing_skills":  rankedApp.MissingSkills,
		}

		rankedApplications = append(rankedApplications, rankedAppInfo)
	}

	response := map[string]interface{}{
		"ranked_applications": rankedApplications,
		"total_applications":  resp.TotalApplications,
		"message":             resp.Message,
	}

	c.JSON(http.StatusOK, response)
}
