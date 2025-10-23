package service

import (
	"context"
	"errors"

	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/models"
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/repository"
	"github.com/google/uuid"
)

type ChatService interface {
	CreateRoom(req *models.CreateRoomRequest, userID string) (*models.Room, error)
	ListRooms() ([]*models.Room, error)
	JoinRoom(roomID, userID string) error
	GetRoomMessages(roomID string, limit int) ([]*models.Message, error)
	SaveMessage(ctx context.Context, msg *models.WebSocketMessage) error
}

type chatService struct {
	roomRepo    repository.RoomRepository
	messageRepo repository.MessageRepository
}

func NewChatService(roomRepo repository.RoomRepository, messageRepo repository.MessageRepository) ChatService {
	return &chatService{
		roomRepo:    roomRepo,
		messageRepo: messageRepo,
	}
}

func (s *chatService) CreateRoom(req *models.CreateRoomRequest, userID string) (*models.Room, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	room := &models.Room{
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   userUUID,
	}

	if err := s.roomRepo.Create(room); err != nil {
		return nil, err
	}

	participant := &models.RoomParticipant{
		RoomID: room.ID,
		UserID: userUUID,
	}

	if err := s.roomRepo.AddParticipant(participant); err != nil {
		return nil, err
	}

	return room, nil
}

func (s *chatService) ListRooms() ([]*models.Room, error) {
	return s.roomRepo.ListAll()
}

func (s *chatService) JoinRoom(roomID, userID string) error {
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return errors.New("invalid room ID")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	isParticipant, err := s.roomRepo.IsParticipant(roomID, userID)
	if err != nil {
		return err
	}

	if isParticipant {
		return nil
	}

	participant := &models.RoomParticipant{
		RoomID: roomUUID,
		UserID: userUUID,
	}

	return s.roomRepo.AddParticipant(participant)
}

func (s *chatService) GetRoomMessages(roomID string, limit int) ([]*models.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.messageRepo.FindByRoomID(roomID, limit)
}

func (s *chatService) SaveMessage(ctx context.Context, msg *models.WebSocketMessage) error {
	roomUUID, err := uuid.Parse(msg.RoomID)
	if err != nil {
		return errors.New("invalid room ID")
	}

	userUUID, err := uuid.Parse(msg.UserID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	message := &models.Message{
		RoomID:   roomUUID,
		UserID:   userUUID,
		Username: msg.Username,
		Content:  msg.Content,
	}

	return s.messageRepo.Create(message)
}
