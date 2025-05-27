package routes

import (
	"context"
	"log"
	"net/http"
	"skillsync-api-gateway/clients"
	"strings"
	"skillsync-api-gateway/utils"
	"github.com/gin-gonic/gin"
	authpb "github.com/shahal0/skillsync-protos/gen/authpb"
	"google.golang.org/grpc/metadata"
)

func SetupRoutes(r *gin.Engine) {
	auth := r.Group("/auth")

	candidate := auth.Group("/candidate")
	{
		candidate.POST("/signup", candidateSignup)
		candidate.POST("/login", candidateLogin)
		candidate.POST("/verify-email", candidateVerifyEmail)
		candidate.POST("/resend-otp", candidateResendOtp)
		candidate.POST("/forgot-password", candidateForgotPassword)
		candidate.PUT("/reset-password", candidateResetPassword)
		candidate.PATCH("/change-password", candidateChangePassword)
		candidate.GET("/profile", candidateProfile)
		candidate.PUT("/profile/update", candidateProfileUpdate)
		candidate.PUT("/Skills/update", candidateSkillsUpdate)
		candidate.PUT("/Education/update", candidateEducationUpdate)
		candidate.POST("/upload/resume", candidateUploadResume)
		candidate.GET("/google/login", candidateGoogleLogin)
		candidate.GET("/google/callback", candidateGoogleCallback)
	}

	employer := auth.Group("/employer")
	{
		employer.POST("/signup", employerSignup)
		employer.POST("/login", employerLogin)
		employer.POST("/verify-email", employerVerifyEmail)
		employer.POST("/resend-otp", employerResendOtp)
		employer.POST("/forgot-password", employerForgotPassword)
		employer.POST("/reset-password", employerResetPassword)
		employer.PATCH("/change-password", employerChangePassword)
		employer.GET("/profile", employerProfile)
		employer.PUT("/profile/update", employerProfileUpdate)
		employer.GET("/google/login", employerGoogleLogin)
		employer.GET("/google/callback", employerGoogleCallback)
	}
}

func candidateSignup(c *gin.Context) {
	var req authpb.CandidateSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Call the CandidateSignup method
	authResp, err := clients.AuthServiceClient.CandidateSignup(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	// Return only id and message as per user preference
	c.JSON(http.StatusOK, authResp)
}

func candidateLogin(c *gin.Context) {
	var req authpb.CandidateLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.CandidateLogin(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	log.Println(resp)
	c.JSON(http.StatusOK, gin.H{
		"id":      resp.Id,
		"message": resp.Message,
		"token":   resp.Token,
	})
}

func candidateVerifyEmail(c *gin.Context) {
	var req authpb.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.CandidateVerifyEmail(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func candidateResendOtp(c *gin.Context) {
	var req authpb.ResendOtpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.CandidateResendOtp(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func candidateForgotPassword(c *gin.Context) {
	var req authpb.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.CandidateForgotPassword(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func candidateResetPassword(c *gin.Context) {
	var req authpb.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.CandidateResetPassword(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func candidateChangePassword(c *gin.Context) {
	var req authpb.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.CandidateChangePassword(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func candidateProfile(c *gin.Context) {
	// Log the request method and path for debugging
	log.Printf("Request: %s %s", c.Request.Method, c.Request.URL.Path)
	token, err := utils.ExtractToken(c)
	if err != nil {
		if err == http.ErrNoCookie {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format, expected 'Bearer token'"})
		}
		return
	}
	log.Printf("Extracted token is empty")

	// Create request with token only
	req := &authpb.CandidateProfileRequest{
		Token: token,
	}

	resp, err := clients.AuthServiceClient.CandidateProfile(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	// Log successful response
	log.Printf("Received successful response from CandidateProfile gRPC method")
	c.JSON(http.StatusOK, resp)
}

func candidateProfileUpdate(c *gin.Context) {
	// Extract token from Authorization header using helper function
	token, err := utils.ExtractToken(c)
	if err != nil {
		if err == http.ErrNoCookie {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format, expected 'Bearer token'"})
		}
		return
	}

	// Parse request body
	var req authpb.CandidateProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set the token from the header
	req.Token = token

	resp, err := clients.AuthServiceClient.CandidateProfileUpdate(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func candidateSkillsUpdate(c *gin.Context) {
	// Extract token from Authorization header using helper function
	token, err := utils.ExtractToken(c)
	if err != nil {
		if err == http.ErrNoCookie {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format, expected 'Bearer token'"})
		}
		return
	}

	// Parse request body
	var req authpb.SkillsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a context with the token in the metadata
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", token),
	)

	// Call the gRPC method with the context containing the token
	resp, err := clients.AuthServiceClient.CandidateSkillsUpdate(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func candidateEducationUpdate(c *gin.Context) {
	// Extract token from Authorization header using helper function
	token, err := utils.ExtractToken(c)
	if err != nil {
		if err == http.ErrNoCookie {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format, expected 'Bearer token'"})
		}
		return
	}

	// Parse request body
	var req authpb.EducationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a context with the token in the metadata
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", token),
	)

	// Call the gRPC method with the context containing the token
	resp, err := clients.AuthServiceClient.CandidateEducationUpdate(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func candidateUploadResume(c *gin.Context) {
	// Extract token from Authorization header using helper function
	token, err := utils.ExtractToken(c)
	if err != nil {
		if err == http.ErrNoCookie {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format, expected 'Bearer token'"})
		}
		return
	}

	// Parse request body
	var req authpb.UploadResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a context with the token in the metadata
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", token),
	)

	// Call the gRPC method with the context containing the token
	resp, err := clients.AuthServiceClient.CandidateUploadResume(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func candidateGoogleLogin(c *gin.Context) {
	// Get the redirect URI from query parameters or use a default one
	redirectURI := c.Query("redirect_uri")
	if redirectURI == "" {
		// Must use the complete URL that's registered in Google Cloud Console
		redirectURI = "http://localhost:8060/candidate/auth/google/callback"
	}
	
	// Log the redirect URI for debugging
	log.Printf("Candidate Google login using redirect URI: %s", redirectURI)
	
	// Create the request with the redirect URI
	req := &authpb.GoogleLoginRequest{	
		RedirectUrl: redirectURI,
	}
	
	// Call the Auth Service to get the Google authorization URL
	resp, err := clients.AuthServiceClient.CandidateGoogleLogin(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	
	// The message field contains the authorization URL
	authURL := resp.GetMessage()
	if authURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Google authorization URL"})
		return
	}
	
	// Redirect the user to the Google authorization URL
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func candidateGoogleCallback(c *gin.Context) {
	// Get the authorization code from the query parameters
	code := c.Query("code")
	
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}
	
	// Create the callback request with the code
	req := &authpb.GoogleCallbackRequest{
		Code: code,
	}
	
	// Call the Auth Service to exchange the code for tokens
	resp, err := clients.AuthServiceClient.CandidateGoogleCallback(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	
	// Check if we got a valid token
	if resp.GetToken() == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate with Google"})
		return
	}
	
	// Set the token as a cookie or return it in the response
	// Option 1: Set as cookie
	c.SetCookie(
		"auth_token",
		resp.GetToken(),
		3600*24, // 24 hours
		"/",
		"", // domain
		true,  // secure
		true,  // httpOnly
	)
	
	// Option 2: Return in response
	c.JSON(http.StatusOK, gin.H{
		"token": resp.GetToken(),
		"message": resp.GetMessage(),
	})
}

func employerSignup(c *gin.Context) {
	var req authpb.EmployerSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.EmployerSignup(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func employerLogin(c *gin.Context) {
	var req authpb.EmployerLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.EmployerLogin(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	// Log the response for debugging
	log.Println("Employer login response:", resp)
	
	// Explicitly include all fields in the response
	c.JSON(http.StatusOK, gin.H{
		"id":      resp.Id,
		"token":   resp.Token,
		"message": resp.Message,
	})
}

func employerVerifyEmail(c *gin.Context) {
	var req authpb.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.EmployerVerifyEmail(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func employerResendOtp(c *gin.Context) {
	var req authpb.ResendOtpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.EmployerResendOtp(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func employerForgotPassword(c *gin.Context) {
	var req authpb.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.EmployerForgotPassword(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func employerResetPassword(c *gin.Context) {
	var req authpb.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.EmployerResetPassword(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func employerChangePassword(c *gin.Context) {
	var req authpb.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := clients.AuthServiceClient.EmployerChangePassword(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func employerProfile(c *gin.Context) {
	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
		return
	}

	// Ensure the token is in the correct format (Bearer token)
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format, expected 'Bearer token'"})
		return
	}

	// Create request with token only
	req := &authpb.EmployerProfileRequest{
		Token: authHeader,
	}

	resp, err := clients.AuthServiceClient.EmployerProfile(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func employerProfileUpdate(c *gin.Context) {
	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
		return
	}

	// Ensure the token is in the correct format (Bearer token)
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format, expected 'Bearer token'"})
		return
	}

	// Parse request body
	var req authpb.EmployerProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set the token from the header
	req.Token = authHeader

	resp, err := clients.AuthServiceClient.EmployerProfileUpdate(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func employerGoogleLogin(c *gin.Context) {
	// Get the redirect URI from query parameters or use a default one
	redirectURI := c.Query("redirect_uri")
	if redirectURI == "" {
		// Must use the complete URL that's registered in Google Cloud Console
		redirectURI = "http://localhost:8060/employer/auth/google/callback"
	}
	
	// Log the redirect URI for debugging
	log.Printf("Employer Google login using redirect URI: %s", redirectURI)
	
	// Create the request with the redirect URI
	req := &authpb.GoogleLoginRequest{	
		RedirectUrl: redirectURI,
	}
	
	// Call the Auth Service to get the Google authorization URL
	resp, err := clients.AuthServiceClient.EmployerGoogleLogin(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	
	// The message field contains the authorization URL
	authURL := resp.GetMessage()
	if authURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Google authorization URL"})
		return
	}
	
	// Redirect the user to the Google authorization URL
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func employerGoogleCallback(c *gin.Context) {
	// Get the authorization code from the query parameters
	code := c.Query("code")
	
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}
	
	// Create the callback request with the code
	req := &authpb.GoogleCallbackRequest{
		Code: code,
	}
	
	// Call the Auth Service to exchange the code for tokens
	resp, err := clients.AuthServiceClient.EmployerGoogleCallback(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	
	// Check if we got a valid token
	if resp.GetToken() == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate with Google"})
		return
	}
	
	c.SetCookie(
		"auth_token",
		resp.GetToken(),
		3600*24, // 24 hours
		"/",
		"", // domain
		true,  // secure
		true,  // httpOnly
	)
	
	// Option 2: Return in response
	c.JSON(http.StatusOK, gin.H{
		"token": resp.GetToken(),
		"message": resp.GetMessage(),
	})
}
