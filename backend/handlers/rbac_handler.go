package handlers

import (
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/example/wg-panel/database"
	"github.com/example/wg-panel/dto"
	"github.com/example/wg-panel/middleware"
	"github.com/example/wg-panel/models"
	"github.com/example/wg-panel/rbac"
	"github.com/example/wg-panel/security"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ===========================================================================
// Users
// ===========================================================================

type UserHandler struct{}

func NewUserHandler() *UserHandler { return &UserHandler{} }

func (h *UserHandler) List(c *gin.Context) {
	allowedSorts := map[string]string{
		"id": "id", "username": "username", "name": "name",
		"active": "active", "created_at": "created_at", "updated_at": "updated_at",
	}
	list := dto.ParseListQuery(c, allowedSorts, "id")
	query := database.DB.Model(&models.User{})
	if list.Search != "" {
		like := "%" + list.Search + "%"
		query = query.Where("username LIKE ? OR name LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to count users")
		return
	}
	var users []models.User
	if err := query.Preload("Roles.Permissions").Preload("Permissions").
		Order(list.OrderClause(allowedSorts)).Limit(list.PerPage).Offset(list.Offset).Find(&users).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch users")
		return
	}
	out := make([]dto.UserResponse, 0, len(users))
	for i := range users {
		out = append(out, userResponse(&users[i]))
	}
	dto.Paginated(c, "data fetched successfully", out, dto.NewListMeta(list, total))
}

func (h *UserHandler) Get(c *gin.Context) {
	user, ok := findUserByParam(c)
	if !ok {
		return
	}
	dto.OK(c, "data fetched successfully", userResponse(user))
}

func (h *UserHandler) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}

	roles, ok := resolveRoles(c, req.RoleIDs)
	if !ok {
		return
	}
	perms, ok := resolvePermissions(c, req.PermissionIDs)
	if !ok {
		return
	}
	hash, err := security.HashPassword(req.Password)
	if err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to hash password")
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	user := models.User{
		Username:     strings.TrimSpace(req.Username),
		Name:         strings.TrimSpace(req.Name),
		PasswordHash: hash,
		Active:       active,
		Roles:        roles,
		Permissions:  perms,
	}
	if err := database.DB.Create(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			dto.Error(c, http.StatusConflict, "username already in use")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to create user")
		return
	}
	reloadUser(&user)
	dto.Created(c, "user created", userResponse(&user))
}

func (h *UserHandler) Update(c *gin.Context) {
	user, ok := findUserByParam(c)
	if !ok {
		return
	}
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}
	// Guard against locking yourself out by disabling your own account.
	if req.Active != nil && !*req.Active && user.ID == middleware.CurrentUserID(c) {
		dto.Error(c, http.StatusConflict, "you cannot disable your own account")
		return
	}
	// Deactivating a super admin must not remove the last active one.
	if req.Active != nil && !*req.Active && !guardLastSuperAdmin(c, user, false) {
		return
	}
	if req.Name != "" {
		user.Name = strings.TrimSpace(req.Name)
	}
	if req.Active != nil {
		user.Active = *req.Active
	}
	if req.Password != "" {
		hash, err := security.HashPassword(req.Password)
		if err != nil {
			dto.Error(c, http.StatusInternalServerError, "failed to hash password")
			return
		}
		user.PasswordHash = hash
	}
	if err := database.DB.Save(user).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to update user")
		return
	}
	reloadUser(user)
	dto.OK(c, "user updated", userResponse(user))
}

func (h *UserHandler) Delete(c *gin.Context) {
	user, ok := findUserByParam(c)
	if !ok {
		return
	}
	if user.ID == middleware.CurrentUserID(c) {
		dto.Error(c, http.StatusConflict, "you cannot delete your own account")
		return
	}
	if !guardLastSuperAdmin(c, user, false) {
		return
	}
	// Clear join rows then hard-delete so pivot tables don't keep orphans and the
	// username is freed for reuse (User has no soft delete).
	if err := database.DB.Select("Roles", "Permissions").Delete(user).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to delete user")
		return
	}
	dto.NoData(c, http.StatusOK, "user deleted")
}

func (h *UserHandler) SetRoles(c *gin.Context) {
	user, ok := findUserByParam(c)
	if !ok {
		return
	}
	var req dto.SetRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	roles, ok := resolveRoles(c, req.RoleIDs)
	if !ok {
		return
	}
	prospective := models.User{Roles: roles, Permissions: user.Permissions}
	if !guardLastSuperAdmin(c, user, prospective.IsSuperAdmin()) {
		return
	}
	if err := database.DB.Model(user).Association("Roles").Replace(roles); err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to set user roles")
		return
	}
	reloadUser(user)
	dto.OK(c, "user roles updated", userResponse(user))
}

func (h *UserHandler) SetPermissions(c *gin.Context) {
	user, ok := findUserByParam(c)
	if !ok {
		return
	}
	var req dto.SetPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	perms, ok := resolvePermissions(c, req.PermissionIDs)
	if !ok {
		return
	}
	prospective := models.User{Roles: user.Roles, Permissions: perms}
	if !guardLastSuperAdmin(c, user, prospective.IsSuperAdmin()) {
		return
	}
	if err := database.DB.Model(user).Association("Permissions").Replace(perms); err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to set user permissions")
		return
	}
	reloadUser(user)
	dto.OK(c, "user direct permissions updated", userResponse(user))
}

// ===========================================================================
// Roles
// ===========================================================================

type RoleHandler struct{}

func NewRoleHandler() *RoleHandler { return &RoleHandler{} }

func (h *RoleHandler) List(c *gin.Context) {
	var roles []models.Role
	if err := database.DB.Preload("Permissions").Order("name asc").Find(&roles).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch roles")
		return
	}
	out := make([]dto.RoleResponse, 0, len(roles))
	for i := range roles {
		out = append(out, roleResponse(&roles[i]))
	}
	dto.OK(c, "data fetched successfully", out)
}

func (h *RoleHandler) Get(c *gin.Context) {
	role, ok := findRoleByParam(c)
	if !ok {
		return
	}
	dto.OK(c, "data fetched successfully", roleResponse(role))
}

func (h *RoleHandler) Create(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}
	perms, ok := resolvePermissions(c, req.PermissionIDs)
	if !ok {
		return
	}
	role := models.Role{Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description), Permissions: perms}
	if err := database.DB.Create(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			dto.Error(c, http.StatusConflict, "role name already in use")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to create role")
		return
	}
	database.DB.Preload("Permissions").First(&role, role.ID)
	dto.Created(c, "role created", roleResponse(&role))
}

func (h *RoleHandler) Update(c *gin.Context) {
	role, ok := findRoleByParam(c)
	if !ok {
		return
	}
	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}
	if role.Name == rbac.SuperAdminRole && strings.TrimSpace(req.Name) != rbac.SuperAdminRole {
		dto.Error(c, http.StatusConflict, "the "+rbac.SuperAdminRole+" role cannot be renamed")
		return
	}
	role.Name = strings.TrimSpace(req.Name)
	role.Description = strings.TrimSpace(req.Description)
	if err := database.DB.Save(role).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			dto.Error(c, http.StatusConflict, "role name already in use")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to update role")
		return
	}
	database.DB.Preload("Permissions").First(role, role.ID)
	dto.OK(c, "role updated", roleResponse(role))
}

func (h *RoleHandler) Delete(c *gin.Context) {
	role, ok := findRoleByParam(c)
	if !ok {
		return
	}
	if !protectSuperAdminRole(c, role, "deleted") {
		return
	}
	if err := database.DB.Select("Permissions").Delete(role).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to delete role")
		return
	}
	dto.NoData(c, http.StatusOK, "role deleted")
}

func (h *RoleHandler) SetPermissions(c *gin.Context) {
	role, ok := findRoleByParam(c)
	if !ok {
		return
	}
	if !protectSuperAdminRole(c, role, "modified") {
		return
	}
	var req dto.SetPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	perms, ok := resolvePermissions(c, req.PermissionIDs)
	if !ok {
		return
	}
	if err := database.DB.Model(role).Association("Permissions").Replace(perms); err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to set role permissions")
		return
	}
	database.DB.Preload("Permissions").First(role, role.ID)
	dto.OK(c, "role permissions updated", roleResponse(role))
}

// ===========================================================================
// Permissions (read-only catalog)
// ===========================================================================

type PermissionHandler struct{}

func NewPermissionHandler() *PermissionHandler { return &PermissionHandler{} }

func (h *PermissionHandler) List(c *gin.Context) {
	var perms []models.Permission
	if err := database.DB.Order("name asc").Find(&perms).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch permissions")
		return
	}
	out := make([]dto.PermissionBrief, 0, len(perms))
	for i := range perms {
		out = append(out, dto.PermissionBrief{ID: perms[i].ID, Name: perms[i].Name, Description: perms[i].Description})
	}
	dto.OK(c, "data fetched successfully", out)
}

// ===========================================================================
// helpers
// ===========================================================================

func findUserByParam(c *gin.Context) (*models.User, bool) {
	id, ok := parseID(c)
	if !ok {
		return nil, false
	}
	var user models.User
	if err := database.DB.Preload("Roles.Permissions").Preload("Permissions").First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "user not found")
		} else {
			dto.Error(c, http.StatusInternalServerError, "failed to fetch user")
		}
		return nil, false
	}
	return &user, true
}

func findRoleByParam(c *gin.Context) (*models.Role, bool) {
	id, ok := parseID(c)
	if !ok {
		return nil, false
	}
	var role models.Role
	if err := database.DB.Preload("Permissions").First(&role, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "role not found")
		} else {
			dto.Error(c, http.StatusInternalServerError, "failed to fetch role")
		}
		return nil, false
	}
	return &role, true
}

// resolveRoles loads the roles for the given IDs, answering 422 if any ID is
// unknown. To prevent privilege escalation, only a super admin may assign a role
// that itself carries the wildcard permission.
func resolveRoles(c *gin.Context, ids []uint) ([]models.Role, bool) {
	ids = dedupeIDs(ids)
	if len(ids) == 0 {
		return []models.Role{}, true
	}
	var roles []models.Role
	if err := database.DB.Preload("Permissions").Where("id IN ?", ids).Find(&roles).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to resolve roles")
		return nil, false
	}
	if len(roles) != len(ids) {
		dto.Error(c, http.StatusUnprocessableEntity, "one or more role_ids do not exist")
		return nil, false
	}
	if !middleware.CurrentUserIsSuperAdmin(c) {
		for i := range roles {
			if roleHasWildcard(&roles[i]) {
				dto.Error(c, http.StatusForbidden, "only a super admin can assign a role that grants the wildcard permission")
				return nil, false
			}
		}
	}
	return roles, true
}

// resolvePermissions loads the permissions for the given IDs, answering 422 if
// any ID is unknown. To prevent privilege escalation, only a super admin may
// grant the wildcard permission directly.
func resolvePermissions(c *gin.Context, ids []uint) ([]models.Permission, bool) {
	ids = dedupeIDs(ids)
	if len(ids) == 0 {
		return []models.Permission{}, true
	}
	var perms []models.Permission
	if err := database.DB.Where("id IN ?", ids).Find(&perms).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to resolve permissions")
		return nil, false
	}
	if len(perms) != len(ids) {
		dto.Error(c, http.StatusUnprocessableEntity, "one or more permission_ids do not exist")
		return nil, false
	}
	if !middleware.CurrentUserIsSuperAdmin(c) {
		for i := range perms {
			if perms[i].Name == models.PermissionWildcard {
				dto.Error(c, http.StatusForbidden, "only a super admin can grant the wildcard permission")
				return nil, false
			}
		}
	}
	return perms, true
}

func roleHasWildcard(role *models.Role) bool {
	for i := range role.Permissions {
		if role.Permissions[i].Name == models.PermissionWildcard {
			return true
		}
	}
	return false
}

// protectSuperAdminRole blocks destructive changes to the seeded super-admin
// role, which would otherwise let a roles.manage holder strip or delete the only
// path to wildcard access (it is re-seeded on every boot regardless).
func protectSuperAdminRole(c *gin.Context, role *models.Role, action string) bool {
	if role.Name == rbac.SuperAdminRole {
		dto.Error(c, http.StatusConflict, "the "+rbac.SuperAdminRole+" role is protected and cannot be "+action)
		return false
	}
	return true
}

// guardLastSuperAdmin writes 409 and returns false if applying a change that
// drops the target user's super-admin status would leave no active super admin.
// stillSuperAfter is whether the user remains an active super admin after it.
func guardLastSuperAdmin(c *gin.Context, user *models.User, stillSuperAfter bool) bool {
	if !user.IsSuperAdmin() || stillSuperAfter {
		return true
	}
	var others []models.User
	if err := database.DB.Preload("Roles.Permissions").Preload("Permissions").
		Where("active = ? AND id <> ?", true, user.ID).Find(&others).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to verify super admins")
		return false
	}
	for i := range others {
		if others[i].IsSuperAdmin() {
			return true
		}
	}
	dto.Error(c, http.StatusConflict, "cannot remove the last active super admin")
	return false
}

func dedupeIDs(ids []uint) []uint {
	seen := make(map[uint]bool, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func reloadUser(user *models.User) {
	database.DB.Preload("Roles.Permissions").Preload("Permissions").First(user, user.ID)
}

func userResponse(u *models.User) dto.UserResponse {
	roles := make([]dto.RoleBrief, 0, len(u.Roles))
	for i := range u.Roles {
		roles = append(roles, dto.RoleBrief{ID: u.Roles[i].ID, Name: u.Roles[i].Name, Description: u.Roles[i].Description})
	}
	direct := make([]dto.PermissionBrief, 0, len(u.Permissions))
	for i := range u.Permissions {
		direct = append(direct, dto.PermissionBrief{ID: u.Permissions[i].ID, Name: u.Permissions[i].Name, Description: u.Permissions[i].Description})
	}
	effSet := u.EffectivePermissions()
	eff := make([]string, 0, len(effSet))
	for name := range effSet {
		eff = append(eff, name)
	}
	sort.Strings(eff)
	return dto.UserResponse{
		ID:                   u.ID,
		Username:             u.Username,
		Name:                 u.Name,
		Active:               u.Active,
		Roles:                roles,
		DirectPermissions:    direct,
		EffectivePermissions: eff,
		CreatedAt:            u.CreatedAt,
		UpdatedAt:            u.UpdatedAt,
	}
}

func roleResponse(r *models.Role) dto.RoleResponse {
	perms := make([]dto.PermissionBrief, 0, len(r.Permissions))
	for i := range r.Permissions {
		perms = append(perms, dto.PermissionBrief{ID: r.Permissions[i].ID, Name: r.Permissions[i].Name, Description: r.Permissions[i].Description})
	}
	return dto.RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Permissions: perms,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
