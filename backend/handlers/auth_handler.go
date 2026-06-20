package handlers

import (
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/example/vpn-manager/auth"
	"github.com/example/vpn-manager/database"
	"github.com/example/vpn-manager/dto"
	"github.com/example/vpn-manager/middleware"
	"github.com/example/vpn-manager/models"
	"github.com/example/vpn-manager/security"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthHandler groups authentication endpoints.
type AuthHandler struct {
	svc *auth.Service
}

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(svc *auth.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Login godoc
// @Summary      Log in
// @Description  Exchange a panel user's credentials for a bearer token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LoginRequest  true  "Login payload"
// @Success      200   {object}  dto.APIResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      422   {object}  dto.ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}

	var user models.User
	err := database.DB.Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusUnauthorized, "invalid username or password")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to look up user")
		return
	}

	valid, verr := security.VerifyPassword(user.PasswordHash, req.Password)
	if verr != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to verify password")
		return
	}
	if !valid {
		dto.Error(c, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if !user.Active {
		dto.Error(c, http.StatusForbidden, "user account is disabled")
		return
	}

	token, expiresAt, err := h.svc.Issue(user.ID, time.Now())
	if err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to issue token")
		return
	}

	dto.OK(c, "login successful", dto.LoginResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresAt: expiresAt.Format(time.RFC3339),
		Username:  user.Username,
	})
}

// Me godoc
// @Summary      Current user
// @Description  Return the authenticated user with roles and effective permissions.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.APIResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		dto.Error(c, http.StatusUnauthorized, "authentication required")
		return
	}
	dto.OK(c, "data fetched successfully", currentUserResponse(user))
}

// currentUserResponse serializes the authenticated user with role names and the
// flattened effective permission set (roles ∪ direct).
func currentUserResponse(u *models.User) gin.H {
	roleNames := make([]string, 0, len(u.Roles))
	for i := range u.Roles {
		roleNames = append(roleNames, u.Roles[i].Name)
	}
	sort.Strings(roleNames)

	perms := u.EffectivePermissions()
	permNames := make([]string, 0, len(perms))
	for name := range perms {
		permNames = append(permNames, name)
	}
	sort.Strings(permNames)

	return gin.H{
		"id":          u.ID,
		"username":    u.Username,
		"name":        u.Name,
		"active":      u.Active,
		"roles":       roleNames,
		"permissions": permNames,
	}
}
