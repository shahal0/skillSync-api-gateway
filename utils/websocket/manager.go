package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket client
type Client struct {
	ID       string
	Role     string // "employer" or "candidate"
	Conn     *websocket.Conn
	Send     chan []byte
	Manager  *Manager
	UserInfo map[string]string // Store additional user info like name, etc.
}

// Manager manages WebSocket connections
type Manager struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	mutex      sync.RWMutex
}

// Message represents a chat message
type Message struct {
	Type           string            `json:"type"`
	SenderID       string            `json:"sender_id"`
	ReceiverID     string            `json:"receiver_id"`
	ConversationID string            `json:"conversation_id"`
	Content        string            `json:"content"`
	SenderRole     string            `json:"sender_role"`
	SentTime       string            `json:"sent_time"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// Global singleton instance of the WebSocket manager
var globalManager *Manager
var managerOnce sync.Once

// NewManager creates a new WebSocket manager
func NewManager() *Manager {
	managerOnce.Do(func() {
		globalManager = &Manager{
			clients:    make(map[string]*Client),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			broadcast:  make(chan *Message),
		}
		// Start the manager in a goroutine
		go globalManager.Start()
	})
	return globalManager
}

// GetManager returns the global WebSocket manager instance
func GetManager() *Manager {
	if globalManager == nil {
		NewManager()
	}
	return globalManager
}

// Start starts the WebSocket manager
func (m *Manager) Start() {
	for {
		select {
		case client := <-m.register:
			m.mutex.Lock()
			m.clients[client.ID] = client
			m.mutex.Unlock()
			log.Printf("Client connected: %s (%s)", client.ID, client.Role)
		
		case client := <-m.unregister:
			if _, ok := m.clients[client.ID]; ok {
				m.mutex.Lock()
				delete(m.clients, client.ID)
				close(client.Send)
				m.mutex.Unlock()
				log.Printf("Client disconnected: %s", client.ID)
			}
		
		case message := <-m.broadcast:
			// Send message to specific user
			if message.ReceiverID != "" {
				m.mutex.RLock()
				if client, ok := m.clients[message.ReceiverID]; ok {
					// Marshal the message to JSON
					jsonMessage, err := json.Marshal(message)
					if err != nil {
						log.Printf("Error marshaling message: %v", err)
						continue
					}
					
					select {
					case client.Send <- jsonMessage:
						log.Printf("Message sent to client %s", client.ID)
					default:
						m.mutex.RUnlock()
						m.mutex.Lock()
						close(client.Send)
						delete(m.clients, client.ID)
						m.mutex.Unlock()
						m.mutex.RLock()
						log.Printf("Client %s removed due to blocked channel", client.ID)
					}
				} else {
					log.Printf("Client %s not found or offline", message.ReceiverID)
				}
				m.mutex.RUnlock()
			}
		}
	}
}

// SendToUser sends a message to a specific user
func (m *Manager) SendToUser(userID string, message *Message) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if client, ok := m.clients[userID]; ok {
		jsonMessage, err := json.Marshal(message)
		if err != nil {
			log.Printf("Error marshaling message: %v", err)
			return
		}
		
		select {
		case client.Send <- jsonMessage:
			log.Printf("Direct message sent to client %s", client.ID)
		default:
			log.Printf("Failed to send message to client %s, channel full", client.ID)
		}
	} else {
		log.Printf("Client %s not found or offline", userID)
	}
}

// GetConnectedUsers returns a list of connected user IDs
func (m *Manager) GetConnectedUsers() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	users := make([]string, 0, len(m.clients))
	for id := range m.clients {
		users = append(users, id)
	}
	
	return users
}

// IsUserConnected checks if a user is connected
func (m *Manager) IsUserConnected(userID string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	_, ok := m.clients[userID]
	return ok
}

// RegisterClient registers a new client with the manager
func (m *Manager) RegisterClient(client *Client) {
	m.register <- client
}
