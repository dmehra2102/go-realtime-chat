package repository

import (
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/models"
	"gorm.io/gorm"
)

type MessageRepository interface {
	Create(message *models.Message) error
	FindByRoomID(roomID string, limit int) ([]*models.Message, error)
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(message *models.Message) error {
	return r.db.Create(message).Error
}

func (r *messageRepository) FindByRoomID(roomID string, limit int) ([]*models.Message, error) {
	var messages []*models.Message
	err := r.db.Where("room_id = ?", roomID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}
