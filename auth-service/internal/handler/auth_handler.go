package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dmehra2102/go-realtime-chat/auth-service/internal/models"
	"github.com/dmehra2102/go-realtime-chat/auth-service/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
	// logger *logger.Logger
}

func NewAuthHandler(authService service.AuthService, // logger *logger.Logger)
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tokenResp, err := h.authService.Register(&req)
	if err != nil {
		// h.logger.Error("Registration failed", "error",err)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, tokenResp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tokenResp, err := h.authService.Login(&req)
	if err != nil {
		// h.logger.Error("Login failed", "error",err)
		h.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, tokenResp)
}

func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, err := h.authService.ValidateToken(req.Token)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	h.respondJSON(w, http.StatusOK, claims)
}

func (h *AuthHandler) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Context-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *AuthHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
