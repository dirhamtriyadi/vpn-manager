package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/example/vpn-manager/auth"
	"github.com/example/vpn-manager/database"
	"github.com/example/vpn-manager/dto"
	"github.com/example/vpn-manager/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Context keys set by Auth for downstream handlers.
const (
	// ContextAuthUser holds the authenticated *models.User.
	ContextAuthUser = "auth_user"
	// ContextAuthPerms holds the user's effective permission set (map[string]bool).
	ContextAuthPerms = "auth_perms"
)

const bearerPrefix = "Bearer "

// Auth guards protected routes: it requires a valid `Authorization: Bearer`
// token, loads the corresponding active user (with roles + direct permissions),
// and stashes the user and its effective permission set on the context.
func Auth(svc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(header, bearerPrefix) {
			dto.Error(c, http.StatusUnauthorized, "authentication required")
			c.Abort()
			return
		}

		token := strings.TrimSpace(header[len(bearerPrefix):])
		userID, err := svc.Validate(token, time.Now())
		if err != nil {
			dto.Error(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		var user models.User
		if err := database.DB.Preload("Roles.Permissions").Preload("Permissions").First(&user, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				dto.Error(c, http.StatusUnauthorized, "user no longer exists")
			} else {
				dto.Error(c, http.StatusInternalServerError, "failed to load user")
			}
			c.Abort()
			return
		}
		if !user.Active {
			dto.Error(c, http.StatusForbidden, "user account is disabled")
			c.Abort()
			return
		}

		c.Set(ContextAuthUser, &user)
		c.Set(ContextAuthPerms, user.EffectivePermissions())
		c.Next()
	}
}

// RequirePermission aborts with 403 unless the authenticated user holds the
// named permission (or the wildcard). It must run after Auth.
func RequirePermission(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		perms, ok := c.Get(ContextAuthPerms)
		if !ok {
			dto.Error(c, http.StatusUnauthorized, "authentication required")
			c.Abort()
			return
		}
		set, _ := perms.(map[string]bool)
		if set[models.PermissionWildcard] || set[perm] {
			c.Next()
			return
		}
		dto.Error(c, http.StatusForbidden, "missing required permission: "+perm)
		c.Abort()
	}
}

// CurrentUser returns the authenticated user set by Auth.
func CurrentUser(c *gin.Context) (*models.User, bool) {
	v, ok := c.Get(ContextAuthUser)
	if !ok {
		return nil, false
	}
	u, ok := v.(*models.User)
	return u, ok
}

// CurrentUserID returns the authenticated user's ID, or 0 if unauthenticated.
func CurrentUserID(c *gin.Context) uint {
	if u, ok := CurrentUser(c); ok {
		return u.ID
	}
	return 0
}

// CurrentUserIsSuperAdmin reports whether the authenticated user holds the
// wildcard permission and so bypasses ownership scoping.
func CurrentUserIsSuperAdmin(c *gin.Context) bool {
	perms, ok := c.Get(ContextAuthPerms)
	if !ok {
		return false
	}
	set, _ := perms.(map[string]bool)
	return set[models.PermissionWildcard]
}
