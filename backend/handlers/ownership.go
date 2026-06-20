package handlers

import (
	"errors"
	"net/http"

	"github.com/example/vpn-manager/database"
	"github.com/example/vpn-manager/dto"
	"github.com/example/vpn-manager/middleware"
	"github.com/example/vpn-manager/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Resource ownership scopes which VPNs each user manages. A super admin (wildcard
// permission) sees and manages everything, including unowned (legacy) rows.
// Every other user is limited to resources whose owner_id is their own user id.
// Non-owned access is reported as 404 so a user cannot probe for resources that
// belong to someone else.

// scopeOwned restricts a query to the current user's owned rows (no-op for super
// admins). The model must have an owner_id column.
func scopeOwned(c *gin.Context, q *gorm.DB) *gorm.DB {
	if middleware.CurrentUserIsSuperAdmin(c) {
		return q
	}
	return q.Where("owner_id = ?", middleware.CurrentUserID(c))
}

// scopeOwnedPeers restricts a peer query to peers whose parent interface is owned
// by the current user (peers inherit their interface's ownership).
func scopeOwnedPeers(c *gin.Context, q *gorm.DB) *gorm.DB {
	if middleware.CurrentUserIsSuperAdmin(c) {
		return q
	}
	ownedIfaceIDs := database.DB.Unscoped().Model(&models.WGInterface{}).
		Select("id").Where("owner_id = ?", middleware.CurrentUserID(c))
	return q.Where("interface_id IN (?)", ownedIfaceIDs)
}

// ownsResource reports whether the current user may act on a resource with the
// given owner. Super admins may act on anything; others only on rows they own.
func ownsResource(c *gin.Context, ownerID *uint) bool {
	if middleware.CurrentUserIsSuperAdmin(c) {
		return true
	}
	return ownerID != nil && *ownerID == middleware.CurrentUserID(c)
}

// currentOwnerID returns the owner id to stamp on a resource at create time.
func currentOwnerID(c *gin.Context) *uint {
	id := middleware.CurrentUserID(c)
	if id == 0 {
		return nil
	}
	return &id
}

// getOwnedInterface loads an interface by id and enforces ownership, answering
// 404 for a missing or not-owned interface and 500 for a real DB error.
func getOwnedInterface(c *gin.Context, id uint) (*models.WGInterface, bool) {
	var iface models.WGInterface
	if err := database.DB.First(&iface, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "interface not found")
		} else {
			dto.Error(c, http.StatusInternalServerError, "failed to fetch interface")
		}
		return nil, false
	}
	if !ownsResource(c, iface.OwnerID) {
		dto.Error(c, http.StatusNotFound, "interface not found")
		return nil, false
	}
	return &iface, true
}

// assertPeerOwned enforces that the current user owns the peer's parent
// interface, answering 404 otherwise (peers inherit interface ownership).
func assertPeerOwned(c *gin.Context, peer *models.Peer) bool {
	if middleware.CurrentUserIsSuperAdmin(c) {
		return true
	}
	var iface models.WGInterface
	if err := database.DB.Unscoped().Select("id", "owner_id").First(&iface, peer.InterfaceID).Error; err != nil {
		dto.Error(c, http.StatusNotFound, "peer not found")
		return false
	}
	if !ownsResource(c, iface.OwnerID) {
		dto.Error(c, http.StatusNotFound, "peer not found")
		return false
	}
	return true
}
