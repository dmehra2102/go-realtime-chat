package hub

import (
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/client"
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/models"
	redispkg "github.com/dmehra2102/go-realtime-chat/chat-service/pkg/redis"
)

type Hub struct {
	clients     map[*client.Client]bool
	rooms       map[string]map[*client.Client]bool
	register    chan *client.Client
	unregister  chan *client.Client
	broadcast   chan *models.WebSocketMessage
	redisPubSub *redispkg.RedisPubSub
	// chatService service
}
