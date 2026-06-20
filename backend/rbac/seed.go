package rbac

import (
	"fmt"
	"log"
	"strings"

	"github.com/example/vpn-manager/models"
	"github.com/example/vpn-manager/security"
	"gorm.io/gorm"
)

// Seed upserts the permission catalog and default roles, then bootstraps the
// first super-admin user from the env-configured credentials when the users
// table is empty. It is idempotent and safe to run on every startup.
//
// When no users exist and adminPassword is empty, no admin is created and the
// API stays locked (login impossible) until a user is provisioned out of band —
// the same "locked until configured" behavior the env-only auth had.
func Seed(db *gorm.DB, adminUsername, adminPassword string) error {
	if err := seedPermissions(db); err != nil {
		return fmt.Errorf("seed permissions: %w", err)
	}
	if err := seedRoles(db); err != nil {
		return fmt.Errorf("seed roles: %w", err)
	}
	if err := bootstrapAdmin(db, adminUsername, adminPassword); err != nil {
		return fmt.Errorf("bootstrap admin: %w", err)
	}
	return nil
}

func seedPermissions(db *gorm.DB) error {
	for _, def := range AllPermissions() {
		perm := models.Permission{Name: def.Name}
		if err := db.Where(models.Permission{Name: def.Name}).
			Assign(models.Permission{Description: def.Description}).
			FirstOrCreate(&perm).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedRoles(db *gorm.DB) error {
	for _, def := range DefaultRoles() {
		role := models.Role{Name: def.Name}
		if err := db.Where(models.Role{Name: def.Name}).
			Assign(models.Role{Description: def.Description}).
			FirstOrCreate(&role).Error; err != nil {
			return err
		}
		var perms []models.Permission
		if err := db.Where("name IN ?", def.Permissions).Find(&perms).Error; err != nil {
			return err
		}
		// Replace keeps the default roles in sync with the catalog on every boot.
		if err := db.Model(&role).Association("Permissions").Replace(perms); err != nil {
			return err
		}
	}
	return nil
}

func bootstrapAdmin(db *gorm.DB, adminUsername, adminPassword string) error {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	username := strings.TrimSpace(adminUsername)
	if username == "" {
		username = "admin"
	}
	if strings.TrimSpace(adminPassword) == "" {
		log.Println("RBAC: no users exist and AUTH_PASSWORD is empty; no admin created. The API is locked until a user is provisioned. Set AUTH_USERNAME/AUTH_PASSWORD and restart to bootstrap a super admin.")
		return nil
	}
	hash, err := security.HashPassword(adminPassword)
	if err != nil {
		return err
	}
	user := models.User{Username: username, Name: "Administrator", PasswordHash: hash, Active: true}
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	var superAdmin models.Role
	if err := db.Where("name = ?", SuperAdminRole).First(&superAdmin).Error; err != nil {
		return err
	}
	if err := db.Model(&user).Association("Roles").Append(&superAdmin); err != nil {
		return err
	}
	log.Printf("RBAC: bootstrapped super-admin user %q from AUTH_USERNAME/AUTH_PASSWORD. Change the password after first login.", username)
	return nil
}
