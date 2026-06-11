package database

import (
	"fmt"
	"log"

	"github.com/example/wg-panel/config"
	"github.com/example/wg-panel/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the shared GORM database handle.
var DB *gorm.DB

// Connect opens a Postgres connection and runs auto-migration.
func Connect(cfg *config.Config) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode,
	)

	gormLogger := logger.Default.LogMode(logger.Info)
	if cfg.GinMode == "release" {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&models.WGInterface{}, &models.Peer{}); err != nil {
		log.Fatalf("failed to run auto migration: %v", err)
	}

	DB = db
	log.Println("database connected and migrated")
}
