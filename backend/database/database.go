package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

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
	//
	// Right after an in-place "docker compose restart", a populated Postgres runs
	// WAL/crash recovery and transiently refuses connections ("the database system
	// is starting up"). Retry with backoff instead of log.Fatal-ing on the first
	// attempt — otherwise the container (which has no restart policy guarantee on
	// every path) exits and stays down until the volume is wiped. A fresh DB has no
	// WAL to replay and connects instantly, which is why this only bit after the
	// data volume had persisted.
	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= 30; attempt++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger, TranslateError: true})
		if err == nil {
			var sqlDB *sql.DB
			if sqlDB, err = db.DB(); err == nil {
				err = sqlDB.Ping()
			}
		}
		if err == nil {
			break
		}
		log.Printf("database not ready (attempt %d/30): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("failed to connect to database after retries: %v", err)
	}

	// Migrate each model on its own so a failure names the offending table. The
	// classic boot-killer here is AutoMigrate building a UNIQUE INDEX that was
	// added to a model after its table already held rows that violate it (e.g. the
	// peer (interface_id, assigned_ip) / public_key indexes added later): on a
	// populated volume CREATE UNIQUE INDEX fails with SQLSTATE 23505 and a generic
	// "failed to run auto migration" hides which table. Naming it makes the fix
	// (dedupe that table) obvious.
	migrations := []struct {
		name  string
		model any
	}{
		{"User", &models.User{}},
		{"Role", &models.Role{}},
		{"Permission", &models.Permission{}},
		{"WGInterface", &models.WGInterface{}},
		{"Peer", &models.Peer{}},
		{"PortForward", &models.PortForward{}},
		{"OpenVPNInstance", &models.OpenVPNInstance{}},
		{"OpenVPNUser", &models.OpenVPNUser{}},
		{"OpenVPNRuntimeManifest", &models.OpenVPNRuntimeManifest{}},
		{"LegacyVPNInstance", &models.LegacyVPNInstance{}},
		{"LegacyVPNUser", &models.LegacyVPNUser{}},
		{"EncryptedSecret", &models.EncryptedSecret{}},
	}
	for _, m := range migrations {
		if err := db.AutoMigrate(m.model); err != nil {
			log.Fatalf("auto migration failed for %s: %v", m.name, err)
		}
	}

	DB = db
	log.Println("database connected and migrated")
}
