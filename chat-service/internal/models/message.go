package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RoomID    uuid.UUID `gorm:"type:uuid;not null;index" json:"room_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Username  string    `gorm:"not null" json:"username"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type WebSocketMessage struct {
	Type     string `json:"type"`
	RoomID   string `json:"room_id,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	Content  string `json:"content,omitempty"`
	Data     any    `json:"data,omitempty"`
}
