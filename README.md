# SkillSync API Gateway

## Overview

The SkillSync API Gateway serves as the central entry point for all client requests to the SkillSync platform. It handles routing, authentication, and communication between clients and the various microservices that make up the SkillSync backend.

## Architecture

The API Gateway follows a microservice architecture pattern, acting as a reverse proxy that:

1. Receives HTTP requests from clients
2. Authenticates and authorizes requests using JWT tokens
3. Routes requests to the appropriate backend service (Auth Service, Job Service, etc.)
4. Transforms responses back to the client

## Features

- **Authentication**: JWT-based authentication with middleware
- **Request Routing**: Routes requests to appropriate microservices
- **Response Transformation**: Formats gRPC responses into JSON
- **Cross-Origin Resource Sharing (CORS)**: Configured for web clients
- **Error Handling**: Consistent error responses

## Services

The API Gateway communicates with the following services:

- **Auth Service**: User authentication, registration, and profile management
- **Job Service**: Job posting, application, and search functionality

## API Endpoints

### Auth Routes

#### Public Routes

- `POST /auth/candidate/signup`: Register a new candidate
- `POST /auth/candidate/login`: Login as a candidate
- `POST /auth/candidate/verify-email`: Verify candidate email
- `POST /auth/candidate/resend-otp`: Resend OTP for verification
- `POST /auth/candidate/forgot-password`: Initiate forgot password flow
- `PUT /auth/candidate/reset-password`: Reset password
- `GET /auth/candidate/google/login`: Google OAuth login for candidates
- `GET /auth/candidate/google/callback`: Google OAuth callback for candidates

- `POST /auth/employer/signup`: Register a new employer
- `POST /auth/employer/login`: Login as an employer
- `POST /auth/employer/verify-email`: Verify employer email
- `POST /auth/employer/resend-otp`: Resend OTP for verification
- `POST /auth/employer/forgot-password`: Initiate forgot password flow
- `PUT /auth/employer/reset-password`: Reset password
- `GET /auth/employer/google/login`: Google OAuth login for employers
- `GET /auth/employer/google/callback`: Google OAuth callback for employers

#### Protected Routes (Require Authentication)

- `PATCH /auth/candidate/change-password`: Change candidate password
- `GET /auth/candidate/profile`: Get candidate profile
- `PUT /auth/candidate/profile/update`: Update candidate profile
- `PUT /auth/candidate/Skills/update`: Update candidate skills
- `PUT /auth/candidate/Education/update`: Update candidate education
- `POST /auth/candidate/upload/resume`: Upload candidate resume

- `PATCH /auth/employer/change-password`: Change employer password
- `GET /auth/employer/profile`: Get employer profile
- `PUT /auth/employer/profile/update`: Update employer profile

### Job Routes

#### Public Routes

- `GET /jobs`: List all jobs with optional filters
- `GET /jobs/get`: Get job details by ID

#### Protected Routes (Require Authentication)

- `POST /jobs/post`: Post a new job (employers only)
- `POST /jobs/apply`: Apply to a job (candidates only)
- `POST /jobs/addskills`: Add skills to a job (employers only)
- `PUT /jobs/status`: Update job status (employers only)
- `GET /jobs/applications`: Get candidate applications (candidates only)
- `GET /jobs/application`: Get application details
- `GET /jobs/filter-applications`: Filter and rank applications (employers only)
- `GET /jobs/applications-by-job`: Get applications for a specific job (employers only)

## Authentication

The API Gateway uses JWT tokens for authentication. Protected routes require a valid JWT token in the Authorization header:

```
Authorization: Bearer <token>
```

The JWT middleware extracts the user ID and role from the token and makes them available to the route handlers.

## Configuration

Environment variables are used for configuration:

- `PORT`: The port on which the API Gateway listens (default: 8008)
- `AUTH_SERVICE_ADDR`: Address of the Auth Service
- `JOB_SERVICE_ADDR`: Address of the Job Service
- `JWT_SECRET`: Secret key for JWT token validation

## Development

### Prerequisites

- Go 1.16+
- Protocol Buffers compiler
- gRPC tools

### Running Locally

1. Clone the repository
2. Set up environment variables in `.env` file
3. Run the API Gateway:

```bash
go run main.go
```

## Profiling

The API Gateway includes built-in profiling capabilities using Go's `pprof` package. The profiling server runs on port 6062.

### Accessing Profiling Data

1. While the service is running, access the profiling interface at: http://localhost:6062/debug/pprof/
2. Available profiles include:
   - CPU profiling: http://localhost:6062/debug/pprof/profile
   - Heap profiling: http://localhost:6062/debug/pprof/heap
   - Goroutine profiling: http://localhost:6062/debug/pprof/goroutine
   - Block profiling: http://localhost:6062/debug/pprof/block
   - Thread creation profiling: http://localhost:6062/debug/pprof/threadcreate

### Using the Go Tool

You can also use the Go tool to analyze profiles:

```bash
# CPU profile (30-second sample)
go tool pprof http://localhost:6062/debug/pprof/profile

# Memory profile
go tool pprof http://localhost:6062/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:6062/debug/pprof/goroutine
```

Once in the pprof interactive mode, you can use commands like `top`, `web`, `list`, etc. to analyze the profile.

## Error Handling

The API Gateway provides consistent error responses in the following format:

```json
{
  "error": "Error message"
}
```

HTTP status codes are used appropriately to indicate the type of error.
