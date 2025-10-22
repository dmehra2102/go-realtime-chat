package models

import (
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	CreatedBy   uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RoomParticipant struct {
	ID       uuid.UUID `gorm:"uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RoomID   uuid.UUID `gorm:"type:uuid;not null;index" json:"room_id"`
	UserID   uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}

type CreateRoomRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=100"`
	Description string `json:"description"`
}
