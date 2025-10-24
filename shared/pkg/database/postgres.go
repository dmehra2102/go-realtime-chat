package database

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/dmehra2102/go-realtime-chat/shared/pkg/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	DB     *gorm.DB
	config config.DatabaseConfig
}

func NewPostgresConnection(config config.DatabaseConfig) (*Database, error) {
	db, err := gorm.Open(postgres.Open(config.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("filed to ping database: %w", err)
	}

	return &Database{
		DB:     db,
		config: config,
	}, nil
}

func (d *Database) MigrateAuthModels() error {
	migrationPath, err := filepath.Abs("./auth-service/internal/database/migrations")
	if err != nil {
		return fmt.Errorf("failed to get migrations path: %w", err)
	}

	migrationPath = filepath.ToSlash(migrationPath)

	m, err := migrate.New(fmt.Sprintf("file://%s", migrationPath), d.config.MigrationDSN())
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		if sourceErr, dbErr := m.Close(); sourceErr != nil || dbErr != nil {
			log.Printf("migration close error: sourceErr=%v, dbErr=%v", sourceErr, dbErr)
		}
	}()

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (d *Database) MigrateChatModels() error {
	migrationPath, err := filepath.Abs("./chat-service/internal/database/migrations")
	if err != nil {
		return fmt.Errorf("failed to get migrations path: %w", err)
	}

	migrationPath = filepath.ToSlash(migrationPath)

	m, err := migrate.New(fmt.Sprintf("file://%s", migrationPath), d.config.MigrationDSN())
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		if sourceErr, dbErr := m.Close(); sourceErr != nil || dbErr != nil {
			log.Printf("migration close error: sourceErr=%v, dbErr=%v", sourceErr, dbErr)
		}
	}()

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
