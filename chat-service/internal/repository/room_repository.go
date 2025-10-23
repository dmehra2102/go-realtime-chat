package repository

import (
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/models"
	"gorm.io/gorm"
)

type RoomRepository interface {
	Create(room *models.Room) error
	FindByID(id string) (*models.Room, error)
	ListAll() ([]*models.Room, error)
	AddParticipant(participant *models.RoomParticipant) error
	IsParticipant(roomID, userID string) (bool, error)
}

type roomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) RoomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) Create(room *models.Room) error {
	return r.db.Create(room).Error
}

func (r *roomRepository) FindByID(id string) (*models.Room, error) {
	var room models.Room
	err := r.db.Where("id = ?", id).First(&room).Error
	return &room, err
}

func (r *roomRepository) ListAll() ([]*models.Room, error) {
	var rooms []*models.Room
	err := r.db.Order("created_at DESC").Find(&rooms).Error
	return rooms, err
}

func (r *roomRepository) AddParticipant(participant *models.RoomParticipant) error {
	return r.db.Create(participant).Error
}

func (r *roomRepository) IsParticipant(roomID, userID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.RoomParticipant{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count).Error
	return count > 0, err
}
