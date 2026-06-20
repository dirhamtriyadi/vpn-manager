package dto

import "time"

// ---- requests ----

type CreateUserRequest struct {
	Username      string `json:"username" validate:"required,min=3,max=64"`
	Name          string `json:"name" validate:"max=128"`
	Password      string `json:"password" validate:"required,min=8,max=128"`
	Active        *bool  `json:"active"`
	RoleIDs       []uint `json:"role_ids"`
	PermissionIDs []uint `json:"permission_ids"`
}

type UpdateUserRequest struct {
	Name     string `json:"name" validate:"max=128"`
	Password string `json:"password" validate:"omitempty,min=8,max=128"`
	Active   *bool  `json:"active"`
}

type SetRolesRequest struct {
	RoleIDs []uint `json:"role_ids"`
}

type SetPermissionsRequest struct {
	PermissionIDs []uint `json:"permission_ids"`
}

type CreateRoleRequest struct {
	Name          string `json:"name" validate:"required,min=2,max=64"`
	Description   string `json:"description" validate:"max=255"`
	PermissionIDs []uint `json:"permission_ids"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=64"`
	Description string `json:"description" validate:"max=255"`
}

// ---- responses ----

type PermissionBrief struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type RoleBrief struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type RoleResponse struct {
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Permissions []PermissionBrief `json:"permissions"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type UserResponse struct {
	ID                   uint              `json:"id"`
	Username             string            `json:"username"`
	Name                 string            `json:"name"`
	Active               bool              `json:"active"`
	Roles                []RoleBrief       `json:"roles"`
	DirectPermissions    []PermissionBrief `json:"direct_permissions"`
	EffectivePermissions []string          `json:"effective_permissions"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}
