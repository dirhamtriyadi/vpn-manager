package database

import (
	"fmt"
	"log"

	"github.com/example/vpn-manager/config"
	"github.com/example/vpn-manager/models"
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

	// TranslateError maps driver-specific errors (e.g. Postgres SQLSTATE 23505)
	// onto GORM's portable sentinels like gorm.ErrDuplicatedKey, so handlers can
	// detect a unique-constraint violation with errors.Is and answer 409 instead
	// of leaking a 500.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger, TranslateError: true})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.WGInterface{},
		&models.Peer{},
		&models.OpenVPNInstance{},
		&models.OpenVPNUser{},
		&models.OpenVPNRuntimeManifest{},
		&models.LegacyVPNInstance{},
		&models.LegacyVPNUser{},
		&models.EncryptedSecret{},
	); err != nil {
		log.Fatalf("failed to run auto migration: %v", err)
	}

	DB = db
	log.Println("database connected and migrated")
}
