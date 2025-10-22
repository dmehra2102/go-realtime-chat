package client

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/models"
	"github.com/dmehra2102/go-realtime-chat/shared/pkg/logger"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	sendBufferSize = 256
)

type Client struct {
	ID       uuid.UUID
	UserID   string
	Username string
	Hub      Hub
	Conn     *websocket.Conn
	Send     chan []byte
	Rooms    map[string]bool
	Logger   *logger.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

type Hub interface {
	Register(client *Client)
	Unregister(client *Client)
	Broadcast(message *models.WebSocketMessage)
}

func NewClient(userID, username string, hub Hub, conn *websocket.Conn, logger *logger.Logger) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		ID:       uuid.New(),
		UserID:   userID,
		Username: username,
		Hub:      hub,
		Conn:     conn,
		Send:     make(chan []byte, sendBufferSize),
		Rooms:    make(map[string]bool),
		Logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (c *Client) Close() {
	c.cancel()
	close(c.Send)
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			var msg models.WebSocketMessage
			err := c.Conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.Logger.Error("WebSocket read error", "error", err)
				}
				return
			}

			msg.UserID = c.UserID
			msg.Username = c.Username

			switch msg.Type {
			case "join":
				c.Rooms[msg.RoomID] = true
				c.Hub.Broadcast(&msg)
			case "leave":
				delete(c.Rooms, msg.RoomID)
				c.Hub.Broadcast(&msg)
			case "message":
				c.Hub.Broadcast(&msg)
			default:
				c.Logger.Warn("Unknown message type", "type", msg.Type)
				continue
			}
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			if c.Conn != nil {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			}
			return
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) SendMessage(msg *models.WebSocketMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		c.Logger.Error("Failed to marshal message", "error", err)
		return
	}

	select {
	case c.Send <- data:
	case <-c.ctx.Done():
	default:
		c.Logger.Warn("Client send buffer full, closing connection")
		c.Close()
	}
}
