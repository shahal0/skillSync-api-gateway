package main

import (
	"skillsync-api-gateway/clients"
	"skillsync-api-gateway/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	clients.InitClients()
	r := gin.Default()
	routes.SetupRoutes(r)
	routes.SetupJobRoutes(r)
	routes.SetupChatNotificationRoutes(r)
	r.Run(":8008")
}
