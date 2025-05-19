package clients

import (
	"google.golang.org/grpc"
	"log"
	"github.com/shahal0/skillsync-protos/gen/authpb"
)

var AuthServiceClient authpb.AuthServiceClient

func InitClients() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to auth-service: %v", err)
	}
	AuthServiceClient = authpb.NewAuthServiceClient(conn)
}

