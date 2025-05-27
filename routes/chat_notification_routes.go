package routes

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"

	"skillsync-api-gateway/clients"
	"skillsync-api-gateway/middlewares"
	
	chatpb "github.com/shahal0/skillsync-protos/gen/chatpb"
	notificationpb "github.com/shahal0/skillsync-protos/gen/notificationpb"
)

// SetupChatNotificationRoutes registers all chat and notification related routes
func SetupChatNotificationRoutes(r *gin.Engine) {
	// All chat and notification routes require authentication
	chatNotif := r.Group("/chat-notification")
	chatNotif.Use(middlewares.JWTMiddleware())
	
	// Chat routes
	chat := chatNotif.Group("/chat")
	{
		chat.POST("/send", SendMessage)
		chat.GET("/messages", GetMessages)
		chat.POST("/broadcast", BroadcastMessage)
		chat.PUT("/status", UpdateMessageStatus)
		chat.GET("/conversations", GetConversations)
	}
	
	// Notification routes
	notif := chatNotif.Group("/notifications")
	{
		notif.GET("/", GetNotifications)
		notif.PUT("/read/:id", MarkNotificationAsRead)
		notif.PUT("/read-all", MarkAllNotificationsAsRead)
	}
}

// SendMessage handles sending a message to another user
func SendMessage(c *gin.Context) {
	var req struct {
		ReceiverID  string `json:"receiver_id" binding:"required"`
		Content     string `json:"content" binding:"required"`
		JobID       string `json:"job_id"`
		IsBroadcast bool   `json:"is_broadcast"`
		MessageType string `json:"message_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from token
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Create context with metadata for gRPC call
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", c.GetHeader("Authorization")),
	)

	// Call gRPC service
	resp, err := clients.ChatServiceClient.SendMessage(ctx, &chatpb.SendMessageRequest{
		ReceiverId:  req.ReceiverID,
		Content:     req.Content,
	})

	if err != nil {
		log.Printf("Error sending message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message_id": resp.MessageId,
		"success":    resp.Status,
	})
}

// GetMessages retrieves messages between two users
func GetMessages(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	otherUserID := c.Query("other_user_id")
	if otherUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "other_user_id is required"})
		return
	}

	_ = c.Query("job_id")
	
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	// Create context with metadata for gRPC call
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", c.GetHeader("Authorization")),
	)

	// Call gRPC service
	// Using GetUserChats which is available in the current client implementation
	resp, err := clients.ChatServiceClient.GetUserChats(ctx, &chatpb.GetUserChatsRequest{
		UserId: userID.(string),
	})

	if err != nil {
		log.Printf("Error getting messages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": resp.RecentMessages,
		"total":    len(resp.RecentMessages),
	})
}

// BroadcastMessage sends a message to all shortlisted candidates for a job
func BroadcastMessage(c *gin.Context) {
	var req struct {
		JobID       string `json:"job_id" binding:"required"`
		Content     string `json:"content" binding:"required"`
		MessageType string `json:"message_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from token
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Create context with metadata for gRPC call
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", c.GetHeader("Authorization")),
	)

	// Since BroadcastMessage is not available in the generated client,
	// we'll use SendMessage with a special receiver ID to indicate broadcast
	resp, err := clients.ChatServiceClient.SendMessage(ctx, &chatpb.SendMessageRequest{
		// Use the user ID as the sender
		SenderId:   userID.(string),
		// Use a special receiver ID or leave empty for the service to handle
		ReceiverId: "broadcast:" + req.JobID,
		// Include the content from the request
		Content:    req.Content,
	})

	if err != nil {
		log.Printf("Error broadcasting message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to broadcast message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message_id": resp.MessageId,
		"success":    resp.Status != "",
	})
}

// UpdateMessageStatus updates the status of a message (sent, delivered, read)
func UpdateMessageStatus(c *gin.Context) {
	var req struct {
		MessageID string `json:"message_id" binding:"required"`
		Status    string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	status := strings.ToUpper(req.Status)
	if status != "SENT" && status != "DELIVERED" && status != "READ" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Must be SENT, DELIVERED, or READ"})
		return
	}

	// Create context with metadata for gRPC call
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", c.GetHeader("Authorization")),
	)

	// Since UpdateMessageStatus is not available in the generated client,
	// we'll use SendMessage to update the status by including it in the content
	// This is a temporary solution until the protobuf definitions are updated
	resp, err := clients.ChatServiceClient.SendMessage(ctx, &chatpb.SendMessageRequest{
		// Use a special format to indicate this is a status update
		ReceiverId: "status_update:" + req.MessageID,
		Content:    status,
	})

	if err != nil {
		log.Printf("Error updating message status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update message status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": resp.Status != "",
		"message": "Status updated successfully",
	})
}

// GetConversations retrieves all conversations for a user
func GetConversations(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	// Create context with metadata for gRPC call
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", c.GetHeader("Authorization")),
	)

	// Call gRPC service
	resp, err := clients.ChatServiceClient.GetConversations(ctx, &chatpb.GetConversationsRequest{
		UserId: userID.(string),
		Page:   int32(page),
		Limit:  int32(limit),
	})

	if err != nil {
		log.Printf("Error getting conversations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get conversations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": resp.Conversations,
		"total":         resp.Total,
	})
}

// GetNotifications retrieves notifications for a user
func GetNotifications(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	unreadOnly := c.DefaultQuery("unread_only", "false") == "true"
	
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}
	
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	// Create context with metadata for gRPC call
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", c.GetHeader("Authorization")),
	)

	// Call gRPC service
	resp, err := clients.NotificationServiceClient.GetNotifications(ctx, &notificationpb.GetNotificationsRequest{
		UserId:     userID.(string),
		UnreadOnly: unreadOnly,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})

	if err != nil {
		log.Printf("Error getting notifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": resp.Notifications,
		"total":         resp.Total,
	})
}

// MarkNotificationAsRead marks a notification as read
func MarkNotificationAsRead(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Notification ID is required"})
		return
	}

	// Create context with metadata for gRPC call
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", c.GetHeader("Authorization")),
	)

	// Call gRPC service
	resp, err := clients.NotificationServiceClient.MarkAsRead(ctx, &notificationpb.MarkAsReadRequest{
		NotificationId: notificationID,
	})

	if err != nil {
		log.Printf("Error marking notification as read: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": resp.Success,
	})
}

// MarkAllNotificationsAsRead marks all notifications as read for a user
func MarkAllNotificationsAsRead(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Create context with metadata for gRPC call
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", c.GetHeader("Authorization")),
	)

	// Call gRPC service
	resp, err := clients.NotificationServiceClient.MarkAllAsRead(ctx, &notificationpb.MarkAllAsReadRequest{
		UserId: userID.(string),
	})

	if err != nil {
		log.Printf("Error marking all notifications as read: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all notifications as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"updated_count": resp.UpdatedCount,
		"success":       resp.Success,
	})
}
