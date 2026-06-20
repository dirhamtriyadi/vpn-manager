package models

import "time"

// User is a panel account. Authentication is by username + Argon2id password
// hash (stored in PasswordHash, never serialized). Authorization is the union of
// the permissions granted by the user's Roles and any Permissions granted
// directly to the user (Spatie-style direct permissions).
//
// Users are hard-deleted (no soft delete): there is no user trash, and a plain
// unique index on Username would otherwise let a deleted account squat its name.
type User struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	Username     string `json:"username" gorm:"size:64;uniqueIndex;not null"`
	Name         string `json:"name" gorm:"size:128"`
	PasswordHash string `json:"-" gorm:"size:255;not null"`
	Active       bool   `json:"active" gorm:"not null;default:true"`

	Roles       []Role       `json:"roles,omitempty" gorm:"many2many:user_roles;constraint:OnDelete:CASCADE;"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:user_permissions;constraint:OnDelete:CASCADE;"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Role bundles permissions and is assigned to users.
type Role struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"size:64;uniqueIndex;not null"`
	Description string       `json:"description" gorm:"size:255"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;constraint:OnDelete:CASCADE;"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Permission is a single named capability such as "interfaces.create". The
// wildcard permission "*" grants everything (super admin).
type Permission struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"size:128;uniqueIndex;not null"`
	Description string `json:"description" gorm:"size:255"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PermissionWildcard grants every permission. A user holding it (via a role or
// directly) is treated as a super admin and bypasses ownership scoping.
const PermissionWildcard = "*"

// EffectivePermissions returns the set of permission names the user holds via
// roles plus direct grants. Roles and Permissions must be preloaded (along with
// Roles.Permissions) for the result to be complete.
func (u *User) EffectivePermissions() map[string]bool {
	set := make(map[string]bool)
	for i := range u.Roles {
		for _, p := range u.Roles[i].Permissions {
			set[p.Name] = true
		}
	}
	for _, p := range u.Permissions {
		set[p.Name] = true
	}
	return set
}

// HasPermission reports whether the user holds the named permission (or the
// wildcard). Requires the same preloads as EffectivePermissions.
func (u *User) HasPermission(name string) bool {
	perms := u.EffectivePermissions()
	return perms[PermissionWildcard] || perms[name]
}

// IsSuperAdmin reports whether the user holds the wildcard permission and so
// bypasses per-resource ownership scoping.
func (u *User) IsSuperAdmin() bool {
	return u.EffectivePermissions()[PermissionWildcard]
}
