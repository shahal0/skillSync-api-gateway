package clients

import (
	"google.golang.org/grpc"
	"log"
	"github.com/shahal0/skillsync-protos/gen/authpb"
	jobpb "github.com/shahal0/skillsync-protos/gen/jobpb"
	"os"
)

var (
	AuthServiceClient authpb.AuthServiceClient
	JobServiceClient  jobpb.JobServiceClient
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Printf("Environment variable %s not set, using default: %s", key, fallback)
	return fallback
}

func InitClients() {
	// Auth Service Client
	authConn, err := grpc.Dial(getEnv("AUTH_SERVICE_URL", "localhost:50051"), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to auth-service: %v", err)
	}
	AuthServiceClient = authpb.NewAuthServiceClient(authConn)

	// Job Service Client
	jobConn, err := grpc.Dial(getEnv("JOB_SERVICE_URL", "localhost:50052"), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to job-service: %v", err)
	}
	JobServiceClient = jobpb.NewJobServiceClient(jobConn)
}

