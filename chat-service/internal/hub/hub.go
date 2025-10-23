package hub

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/client"
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/models"
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/service"
	redispkg "github.com/dmehra2102/go-realtime-chat/chat-service/pkg/redis"
	"github.com/dmehra2102/go-realtime-chat/shared/pkg/logger"
)

type Hub struct {
	clients     map[*client.Client]bool
	rooms       map[string]map[*client.Client]bool
	register    chan *client.Client
	unregister  chan *client.Client
	broadcast   chan *models.WebSocketMessage
	redisPubSub *redispkg.RedisPubSub
	chatService service.ChatService
	logger      *logger.Logger
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewHub(redisPubSub *redispkg.RedisPubSub, chatService service.ChatService, logger *logger.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &Hub{
		clients:     make(map[*client.Client]bool),
		rooms:       make(map[string]map[*client.Client]bool),
		register:    make(chan *client.Client),
		unregister:  make(chan *client.Client),
		broadcast:   make(chan *models.WebSocketMessage),
		redisPubSub: redisPubSub,
		chatService: chatService,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
	// Subscribe to Redis messages
	go h.subscribeToRedis()
	return h
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.ctx.Done():
			h.logger.Info("Hub shutting down")
			return
		case client := <-h.register:
			h.handleRegister(client)
		case client := <-h.unregister:
			h.handleUnregister(client)
		case message := <-h.broadcast:
			h.handleBroadcast(message)
		}
	}
}

func (h *Hub) Register(client *client.Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *client.Client) {
	h.unregister <- client
}

func (h *Hub) Broadcast(message *models.WebSocketMessage) {
	h.broadcast <- message
}

func (h *Hub) handleRegister(client *client.Client) {
	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()

	h.logger.Info("Client registered", "userID", client.UserID, "username", client.Username)
}

func (h *Hub) handleUnregister(client *client.Client) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)

		for roomID := range client.Rooms {
			if room, exists := h.rooms[roomID]; exists {
				delete(room, client)
				if len(room) == 0 {
					delete(h.rooms, roomID)
				}

				// Broadcast leave message
				leaveMsg := &models.WebSocketMessage{
					Type:     "leave",
					RoomID:   roomID,
					UserID:   client.UserID,
					Username: client.Username,
				}
				h.broadcastToRoom(roomID, leaveMsg, client)
			}
		}

		client.Close()
	}
	h.mu.Unlock()

	h.logger.Info("Client unregistered", "userID", client.UserID)
}

func (h *Hub) handleBroadcast(message *models.WebSocketMessage) {
	switch message.Type {
	case "join":
		h.handleJoinRoom(message)
	case "leave":
		h.handleLeaveRoom(message)
	case "message":
		h.handleMessage(message)
	}
}

func (h *Hub) handleJoinRoom(message *models.WebSocketMessage) {
	h.mu.Lock()

	var targetClient *client.Client
	for c := range h.clients {
		if c.UserID == message.UserID {
			targetClient = c
			break
		}
	}

	if targetClient != nil {
		if h.rooms[message.RoomID] == nil {
			h.rooms[message.RoomID] = make(map[*client.Client]bool)
		}
		h.rooms[message.RoomID][targetClient] = true
		targetClient.Rooms[message.RoomID] = true
	}

	h.mu.Unlock()

	h.broadcastToRoom(message.RoomID, message, targetClient)

	h.publishToRedis(message)
	h.logger.Info("User joined room", "userID", message.UserID, "roomID", message.RoomID)
}

func (h *Hub) handleLeaveRoom(message *models.WebSocketMessage) {
	h.mu.Lock()

	var targetClient *client.Client
	for c := range h.clients {
		if c.UserID == message.UserID {
			targetClient = c
			break
		}
	}

	if targetClient != nil {
		if room, exists := h.rooms[message.RoomID]; exists {
			delete(room, targetClient)
			if len(room) == 0 {
				delete(h.rooms, message.RoomID)
			}
		}
		delete(targetClient.Rooms, message.RoomID)
	}

	h.mu.Unlock()

	h.broadcastToRoom(message.RoomID, message, targetClient)

	h.publishToRedis(message)

	h.logger.Info("User left room", "userID", message.UserID, "roomID", message.RoomID)
}

func (h *Hub) handleMessage(message *models.WebSocketMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.chatService.SaveMessage(ctx, message); err != nil {
		h.logger.Error("Failed to save message", "error", err)
	}

	h.broadcastToRoom(message.RoomID, message, nil)

	h.publishToRedis(message)
}

func (h *Hub) broadcastToRoom(roomID string, message *models.WebSocketMessage, except *client.Client) {
	h.mu.Lock()
	room, exists := h.rooms[roomID]
	h.mu.Unlock()

	if !exists {
		return
	}
	for client := range room {
		if client != except {
			client.SendMessage(message)
		}
	}
}

func (h *Hub) publishToRedis(message *models.WebSocketMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("Failed to marshal message for Redis", "error", err)
		return
	}

	if err := h.redisPubSub.Publish(h.ctx, "chat.messages", string(data)); err != nil {
		h.logger.Error("Failed to publish to Redis", "error", err)
	}
}

func (h *Hub) subscribeToRedis() {
	msgChan := h.redisPubSub.Subscribe(h.ctx, "chat.messages")

	for {
		select {
		case <-h.ctx.Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				h.logger.Warn("Redis message channel closed")
				return
			}

			var wsMsg models.WebSocketMessage
			if err := json.Unmarshal([]byte(msg), &wsMsg); err != nil {
				h.logger.Error("Failed to unmarshal Redis message", "error", err)
				continue
			}

			h.broadcastToRoom(wsMsg.RoomID, &wsMsg, nil)
		}
	}
}

func (h *Hub) Shutdown() {
	h.cancel()

	h.mu.Lock()
	for client := range h.clients {
		client.Close()
	}
	h.mu.Unlock()

	h.logger.Info("Hub shutdown complete")
}
