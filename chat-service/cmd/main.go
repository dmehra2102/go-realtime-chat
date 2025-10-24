package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/handler"
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/hub"
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/repository"
	"github.com/dmehra2102/go-realtime-chat/chat-service/internal/service"
	redispkg "github.com/dmehra2102/go-realtime-chat/chat-service/pkg/redis"
	"github.com/dmehra2102/go-realtime-chat/shared/pkg/config"
	"github.com/dmehra2102/go-realtime-chat/shared/pkg/database"
	"github.com/dmehra2102/go-realtime-chat/shared/pkg/logger"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load ENV
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	appLogger := logger.NewLogger("chat-service")

	cfg := config.LoadConfig()

	db, err := database.NewPostgresConnection(cfg.Database)
	if err != nil {
		appLogger.Fatal("Failed to connect to database", "error", err)
	}

	if err := db.MigrateChatModels(); err != nil {
		appLogger.Fatal("Failed to migrate database", "error", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		appLogger.Fatal("Failed to connect to Redis", "error", err)
	}

	roomRepo := repository.NewRoomRepository(db.DB)
	messageRepo := repository.NewMessageRepository(db.DB)

	chatService := service.NewChatService(roomRepo, messageRepo)

	redisPubSub := redispkg.NewRedisPubSub(redisClient, appLogger)

	chatHub := hub.NewHub(redisPubSub, chatService, appLogger)
	go chatHub.Run()

	wsHandler := handler.NewWebSocketHandler(chatHub, chatService, cfg.JWTSecret, appLogger)

	router := mux.NewRouter()
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")
	router.HandleFunc("/ws", wsHandler.HandleWebSocket)
	router.HandleFunc("/api/rooms", wsHandler.CreateRoom).Methods("POST")
	router.HandleFunc("/api/rooms", wsHandler.ListRooms).Methods("GET")
	router.HandleFunc("/api/rooms/{roomId}/messages", wsHandler.GetRoomMessages).Methods("GET")

	srv := &http.Server{
		Addr:         ":" + cfg.ChatServicePort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		appLogger.Info("Chat Service starting", "port", cfg.ChatServicePort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down chat service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown", "error", err)
	}

	if err := redisClient.Close(); err != nil {
		appLogger.Error("Failed to close Redis connection", "error", err)
	}

	sqlDB, _ := db.DB.DB()
	if err := sqlDB.Close(); err != nil {
		appLogger.Error("Failed to close database connection", "error", err)
	}

	appLogger.Info("Chat service stopped")
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}
