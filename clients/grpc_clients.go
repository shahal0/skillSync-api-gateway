package clients

import (
	"fmt"
	"google.golang.org/grpc"
	"log"
	"os"
	"github.com/shahal0/skillsync-protos/gen/authpb"
	chatpb "github.com/shahal0/skillsync-protos/gen/chatpb"
	jobpb "github.com/shahal0/skillsync-protos/gen/jobpb"
	notificationpb "github.com/shahal0/skillsync-protos/gen/notificationpb"
)

var (
	AuthServiceClient         authpb.AuthServiceClient
	JobServiceClient          jobpb.JobServiceClient
	ChatServiceClient         chatpb.ChatServiceClient
	NotificationServiceClient notificationpb.NotificationServiceClient
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Printf("Environment variable %s not set, using default: %s", key, fallback)
	return fallback
}

// GetChatClient returns the chat service client
func GetChatClient() (chatpb.ChatServiceClient, error) {
	if ChatServiceClient == nil {
		return nil, fmt.Errorf("chat service client not initialized")
	}
	return ChatServiceClient, nil
}

// GetNotificationClient returns the notification service client
func GetNotificationClient() notificationpb.NotificationServiceClient {
	return NotificationServiceClient
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
	chatNotifConn, err := grpc.Dial(getEnv("CHAT_NOTIFICATION_SERVICE_URL", "localhost:50053"), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to chat-notification-service: %v", err)
	}
	ChatServiceClient = chatpb.NewChatServiceClient(chatNotifConn)
	NotificationServiceClient = notificationpb.NewNotificationServiceClient(chatNotifConn)
}

