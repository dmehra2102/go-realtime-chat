package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmehra2102/go-realtime-chat/auth-service/internal/handler"
	"github.com/dmehra2102/go-realtime-chat/auth-service/internal/repository"
	"github.com/dmehra2102/go-realtime-chat/auth-service/internal/service"
	"github.com/dmehra2102/go-realtime-chat/shared/pkg/config"
	"github.com/dmehra2102/go-realtime-chat/shared/pkg/database"
	"github.com/dmehra2102/go-realtime-chat/shared/pkg/logger"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	appLogger := logger.NewLogger("auth-service")

	cfg := config.LoadConfig()

	// Connect to Database
	db,err := database.NewPostgresConnection(cfg.Database)
	if err != nil {
		appLogger.Fatal("Failed to connect to database", "error", err)
	}

	if err := db.MigrateAuthModels(); err != nil {
		appLogger.Fatal("Failed to migrate database", "error", err)
	}

	userRepo := repository.NewUserRepository(db.DB)

	authService := service.NewAuthService(userRepo, cfg.JWTSecret)

	authHandler := handler.NewAuthHandler(authService, appLogger)

	router := mux.NewRouter()
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")
	router.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST")
	router.HandleFunc("/api/auth/validate", authHandler.ValidateToken).Methods("POSt")

	// Create Server
	srv := &http.Server{
		Addr: ":" + cfg.AuthServicePort,
		Handler: router,
		ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout: 15 * time.Second,
	}

	go func(){
		appLogger.Info("Auth Service starting", "port", cfg.AuthServicePort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<- quit

	appLogger.Info("Shutting down auth service...")

	ctx,cancel := context.WithTimeout(context.Background(), 30 * time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown", "error", err)
	}

	sqlDB, _ := db.DB.DB()
	if err := sqlDB.Close(); err != nil {
		appLogger.Error("Failed to close database connection", "error", err)
	}

	appLogger.Info("Auth service stopped")
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}