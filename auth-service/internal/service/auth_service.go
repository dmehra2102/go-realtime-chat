package service

import (
	"errors"
	"time"

	"github.com/dmehra2102/go-realtime-chat/auth-service/internal/models"
	"github.com/dmehra2102/go-realtime-chat/auth-service/internal/repository"
	"github.com/dmehra2102/go-realtime-chat/auth-service/pkg/hash"
	"github.com/dmehra2102/go-realtime-chat/auth-service/pkg/jwt"
)

type AuthService interface {
	Register(req *models.RegisterRequest) (*models.TokenResponse, error)
	Login(req *models.LoginRequest) (*models.TokenResponse, error)
	ValidateToken(token string) (*jwt.Claims, error)
}

type authService struct {
	userRepo  repository.UserRepository
	jwtSecret string
}

func NewAuthService(userRepo repository.UserRepository, jwtSecret string) AuthService {
	return &authService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (s *authService) Register(req *models.RegisterRequest) (*models.TokenResponse, error) {
	if _, err := s.userRepo.FindByEmail(req.Email); err == nil {
		return nil, errors.New("user with this email already exists")
	}

	if _, err := s.userRepo.FindByUsername(req.Username); err == nil {
		return nil, errors.New("username already taken")
	}

	passwordHash, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	newUser := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
	}

	if err := s.userRepo.Create(newUser); err != nil {
		return nil, errors.New("failed to create user")
	}

	return s.generateTokenResponse(newUser)
}

func (s *authService) Login(req *models.LoginRequest) (*models.TokenResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !hash.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	return s.generateTokenResponse(user)
}

func (s *authService) ValidateToken(token string) (*jwt.Claims, error) {
	return jwt.ValidateToken(token, s.jwtSecret)
}

func (s *authService) generateTokenResponse(user *models.User) (*models.TokenResponse, error) {
	expiresIn := 24 * time.Hour
	accessToken, err := jwt.GenerateToken(user.ID.String(), user.Username, expiresIn, s.jwtSecret)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	refreshToken, err := jwt.GenerateToken(user.ID.String(), user.Username, 7*24*time.Hour, s.jwtSecret)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(expiresIn.Seconds()),
		User: models.UserDTO{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		},
	}, nil
}
