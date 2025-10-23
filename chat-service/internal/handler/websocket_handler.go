package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dmehra2102/go-realtime-chat/auth-service/pkg/jwt"
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/client"
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/hub"
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/models"
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/service"
	"github.com/dmehra2102/go-realtime-chat/shared/pkg/logger"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// allowedOrigins := map[string]bool{
		// 	"https://yourdomain.com": true,
		// 	"https://app.yourdomain.com": true,
		// }
		// return allowedOrigins[r.Header.Get("Origin")]
		return true
	},
}

type WebSocketHandler struct {
	hub         *hub.Hub
	chatService service.ChatService
	jwtSecret   string
	logger      *logger.Logger
}

func NewWebSocketHandler(hub *hub.Hub, chatService service.ChatService, jwtSecret string, logger *logger.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:         hub,
		chatService: chatService,
		jwtSecret:   jwtSecret,
		logger:      logger,
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		token = extractTokenFromHeader(r.Header.Get("Authorization"))
	}

	if token == "" {
		http.Error(w, "Unauthorized: No token provided", http.StatusUnauthorized)
		return
	}

	claims, err := jwt.ValidateToken(token, h.jwtSecret)
	if err != nil {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection", "error", err)
		return
	}

	newClient := client.NewClient(claims.UserID, claims.Username, h.hub, conn, h.logger)

	// Register a client
	h.hub.Register(newClient)

	// Start goroutines
	go newClient.WritePump()
	go newClient.ReadPump()
}

func (h *WebSocketHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	token := extractTokenFromHeader(r.Header.Get("Authorization"))
	claims, err := jwt.ValidateToken(token, h.jwtSecret)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	room, err := h.chatService.CreateRoom(&req, claims.UserID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to create room")
		return
	}

	h.respondJSON(w, http.StatusCreated, room)
}

func (h *WebSocketHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.chatService.ListRooms()
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch rooms")
		return
	}

	h.respondJSON(w, http.StatusOK, rooms)
}

func (h *WebSocketHandler) GetRoomMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["roomID"]

	messages, err := h.chatService.GetRoomMessages(roomID, 50)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch messages")
		return
	}

	h.respondJSON(w, http.StatusOK, messages)
}

func (h *WebSocketHandler) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *WebSocketHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

func extractTokenFromHeader(authHeader string) string {
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	return " "
}
