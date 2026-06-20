package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/example/vpn-manager/database"
	"github.com/example/vpn-manager/dto"
	"github.com/example/vpn-manager/middleware"
	"github.com/example/vpn-manager/models"
	"github.com/example/vpn-manager/wg"
	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

// PeerHandler groups peer endpoints.
type PeerHandler struct{}

// NewPeerHandler builds a PeerHandler.
func NewPeerHandler() *PeerHandler { return &PeerHandler{} }

// List godoc
// @Summary      List peers of an interface
// @Tags         peers
// @Produce      json
// @Param        id   path      int  true  "Interface ID"
// @Success      200  {object}  dto.APIResponse
// @Router       /interfaces/{id}/peers [get]
func (h *PeerHandler) List(c *gin.Context) {
	ifaceID, ok := parseID(c)
	if !ok {
		return
	}
	if _, ok := getOwnedInterface(c, ifaceID); !ok {
		return
	}
	allowedSorts := map[string]string{
		"id":          "id",
		"name":        "name",
		"assigned_ip": "assigned_ip",
		"enabled":     "enabled",
		"created_at":  "created_at",
		"updated_at":  "updated_at",
	}
	list := dto.ParseListQuery(c, allowedSorts, "id")
	query := database.DB.Model(&models.Peer{}).Where("interface_id = ?", ifaceID)
	if list.Search != "" {
		like := "%" + list.Search + "%"
		query = query.Where("name LIKE ? OR assigned_ip LIKE ? OR allowed_ips LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to count peers")
		return
	}

	var peers []models.Peer
	if err := query.Order(list.OrderClause(allowedSorts)).Limit(list.PerPage).Offset(list.Offset).Find(&peers).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch peers")
		return
	}
	dto.Paginated(c, "data fetched successfully", peers, dto.NewListMeta(list, total))
}

// Create godoc
// @Summary      Add a peer
// @Description  Server auto-generates keys and IP when omitted
// @Tags         peers
// @Accept       json
// @Produce      json
// @Param        id    path      int                   true  "Interface ID"
// @Param        body  body      dto.CreatePeerRequest true  "Peer payload"
// @Success      201   {object}  dto.APIResponse
// @Failure      422   {object}  dto.ErrorResponse
// @Router       /interfaces/{id}/peers [post]
func (h *PeerHandler) Create(c *gin.Context) {
	ifaceID, ok := parseID(c)
	if !ok {
		return
	}

	var iface models.WGInterface
	if err := database.DB.Preload("Peers").First(&iface, ifaceID).Error; err != nil {
		respondInterfaceLookupError(c, err)
		return
	}
	if !ownsResource(c, iface.OwnerID) {
		dto.Error(c, http.StatusNotFound, "interface not found")
		return
	}

	var req dto.CreatePeerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}

	peer := models.Peer{
		InterfaceID:         iface.ID,
		Name:                req.Name,
		PublicKey:           req.PublicKey,
		ClientAllowedIPs:    defaultString(req.ClientAllowedIPs, "0.0.0.0/0, ::/0"),
		PersistentKeepalive: req.PersistentKeepalive,
		Enabled:             true,
	}
	if req.Enabled != nil {
		peer.Enabled = *req.Enabled
	}

	// Generate a key pair when the client did not bring its own public key.
	if peer.PublicKey == "" {
		kp, err := wg.GenerateKeyPair()
		if err != nil {
			dto.Error(c, http.StatusInternalServerError, "failed to generate keys")
			return
		}
		peer.PrivateKey = kp.PrivateKey
		peer.PublicKey = kp.PublicKey
	}
	if err := wg.ValidatePublicKey(peer.PublicKey); err != nil {
		dto.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := wg.ValidateCIDRList(peer.ClientAllowedIPs); err != nil {
		dto.Error(c, http.StatusUnprocessableEntity, "invalid client_allowed_ips: "+err.Error())
		return
	}

	if req.UsePresharedKey {
		psk, err := wg.GeneratePresharedKey()
		if err != nil {
			dto.Error(c, http.StatusInternalServerError, "failed to generate preshared key")
			return
		}
		peer.PresharedKey = psk
	}

	// Assign IP (explicit or next free in the subnet).
	assigned := req.AssignedIP
	if assigned == "" {
		taken := make([]string, 0, len(iface.Peers))
		for _, p := range iface.Peers {
			taken = append(taken, p.AssignedIP)
		}
		next, err := wg.NextFreeIP(iface.Address, taken)
		if err != nil {
			dto.Error(c, http.StatusUnprocessableEntity, err.Error())
			return
		}
		assigned = next
	}
	peer.AssignedIP = assigned
	peer.AllowedIPs = assigned + "/32"
	if err := wg.ValidateIPInCIDR(peer.AssignedIP, iface.Address); err != nil {
		dto.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if conflict, err := ensurePeerUnique(peer); err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to check peer uniqueness")
		return
	} else if conflict != "" {
		dto.Error(c, http.StatusConflict, conflict)
		return
	}

	if err := database.DB.Create(&peer).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			dto.Error(c, http.StatusConflict, "peer public key or assigned IP already in use")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to create peer")
		return
	}

	if err := syncPeer(iface.ID, peer, false); err != nil {
		dto.CreatedWarn(c, "peer created", peer, "not applied to kernel: "+err.Error())
		return
	}
	dto.Created(c, "peer created", peer)
}

// Update godoc
// @Summary      Update a peer
// @Tags         peers
// @Accept       json
// @Produce      json
// @Param        peerId  path      int                   true  "Peer ID"
// @Param        body    body      dto.UpdatePeerRequest true  "Peer payload"
// @Success      200     {object}  dto.APIResponse
// @Failure      404     {object}  dto.ErrorResponse
// @Router       /peers/{peerId} [put]
func (h *PeerHandler) Update(c *gin.Context) {
	peer, err := findPeer(c)
	if err != nil {
		return
	}
	if !assertPeerOwned(c, peer) {
		return
	}

	var req dto.UpdatePeerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}

	peer.Name = req.Name
	if req.ClientAllowedIPs != "" {
		peer.ClientAllowedIPs = req.ClientAllowedIPs
	}
	if err := wg.ValidateCIDRList(peer.ClientAllowedIPs); err != nil {
		dto.Error(c, http.StatusUnprocessableEntity, "invalid client_allowed_ips: "+err.Error())
		return
	}
	peer.PersistentKeepalive = req.PersistentKeepalive
	if req.Enabled != nil {
		peer.Enabled = *req.Enabled
	}

	if err := database.DB.Save(peer).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			dto.Error(c, http.StatusConflict, "peer public key or assigned IP already in use")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to update peer")
		return
	}

	if err := syncPeer(peer.InterfaceID, *peer, false); err != nil {
		dto.OKWarn(c, "peer updated", peer, "not applied to kernel: "+err.Error())
		return
	}
	dto.OK(c, "peer updated", peer)
}

// Delete godoc
// @Summary      Delete a peer
// @Tags         peers
// @Produce      json
// @Param        peerId  path      int  true  "Peer ID"
// @Success      200     {object}  dto.APIResponse
// @Failure      404     {object}  dto.ErrorResponse
// @Router       /peers/{peerId} [delete]
func (h *PeerHandler) Delete(c *gin.Context) {
	peer, err := findPeer(c)
	if err != nil {
		return
	}
	if !assertPeerOwned(c, peer) {
		return
	}
	ifaceID := peer.InterfaceID
	peerSnapshot := *peer
	removePortForwardRulesForPeer(interfaceNameByID(peer.InterfaceID), peer.ID)
	if err := database.DB.Delete(peer).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to move peer to trash")
		return
	}
	if err := syncPeer(ifaceID, peerSnapshot, true); err != nil {
		dto.NoDataWarn(c, "peer moved to trash", "kernel not updated: "+err.Error())
		return
	}
	dto.NoData(c, http.StatusOK, "peer moved to trash")
}

func (h *PeerHandler) Trash(c *gin.Context) {
	allowedSorts := map[string]string{
		"id":           "id",
		"name":         "name",
		"interface_id": "interface_id",
		"assigned_ip":  "assigned_ip",
		"created_at":   "created_at",
		"updated_at":   "updated_at",
		"deleted_at":   "deleted_at",
	}
	list := dto.ParseListQuery(c, allowedSorts, "deleted_at")
	if c.Query("sort_order") == "" {
		list.SortOrder = "desc"
	}
	query := scopeOwnedPeers(c, database.DB.Unscoped().Model(&models.Peer{}).Where("deleted_at IS NOT NULL"))
	if list.Search != "" {
		like := "%" + list.Search + "%"
		query = query.Where("name LIKE ? OR assigned_ip LIKE ? OR allowed_ips LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to count trashed peers")
		return
	}

	var peers []models.Peer
	if err := query.Order(list.OrderClause(allowedSorts)).Limit(list.PerPage).Offset(list.Offset).Find(&peers).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch trashed peers")
		return
	}
	dto.Paginated(c, "data fetched successfully", peers, dto.NewListMeta(list, total))
}

func (h *PeerHandler) Restore(c *gin.Context) {
	peer, err := findPeerUnscoped(c)
	if err != nil {
		return
	}
	if !assertPeerOwned(c, peer) {
		return
	}
	if !peer.DeletedAt.Valid {
		dto.OK(c, "peer already active", peer)
		return
	}
	var iface models.WGInterface
	if err := database.DB.First(&iface, peer.InterfaceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusConflict, "restore the parent interface before restoring this peer")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to fetch parent interface")
		return
	}
	if err := database.DB.Unscoped().Model(peer).Update("deleted_at", nil).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to restore peer")
		return
	}
	if err := syncPeer(peer.InterfaceID, *peer, false); err != nil {
		dto.OKWarn(c, "peer restored", peer, "kernel not updated: "+err.Error())
		return
	}
	reapplyPortForwardsForPeer(iface.Name, peer.ID)
	dto.OK(c, "peer restored", peer)
}

func (h *PeerHandler) Purge(c *gin.Context) {
	peer, err := findPeerUnscoped(c)
	if err != nil {
		return
	}
	if !assertPeerOwned(c, peer) {
		return
	}
	peerSnapshot := *peer
	purgePortForwardsForPeer(interfaceNameByID(peer.InterfaceID), peer.ID)
	if err := database.DB.Unscoped().Delete(peer).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to permanently delete peer")
		return
	}
	if err := syncPeer(peerSnapshot.InterfaceID, peerSnapshot, true); err != nil {
		dto.NoDataWarn(c, "peer permanently deleted", "kernel not updated: "+err.Error())
		return
	}
	dto.NoData(c, http.StatusOK, "peer permanently deleted")
}

// Config godoc
// @Summary      Get client config
// @Description  Returns the wg-quick .conf for this peer as plain text
// @Tags         peers
// @Produce      plain
// @Param        peerId  path      int  true  "Peer ID"
// @Success      200     {string}  string
// @Failure      404     {object}  dto.ErrorResponse
// @Router       /peers/{peerId}/config [get]
func (h *PeerHandler) Config(c *gin.Context) {
	peer, err := findPeer(c)
	if err != nil {
		return
	}
	if !assertPeerOwned(c, peer) {
		return
	}
	var iface models.WGInterface
	if err := database.DB.First(&iface, peer.InterfaceID).Error; err != nil {
		respondInterfaceLookupError(c, err)
		return
	}
	cfg := wg.BuildClientConfig(&iface, peer)
	c.Header("Content-Disposition", "attachment; filename=\""+peer.Name+".conf\"")
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(cfg))
}

// QRCode godoc
// @Summary      Get client config QR code
// @Description  Returns a PNG QR code of the peer's wg-quick config
// @Tags         peers
// @Produce      png
// @Param        peerId  path      int  true  "Peer ID"
// @Success      200     {file}    binary
// @Failure      404     {object}  dto.ErrorResponse
// @Router       /peers/{peerId}/qrcode [get]
func (h *PeerHandler) QRCode(c *gin.Context) {
	peer, err := findPeer(c)
	if err != nil {
		return
	}
	if !assertPeerOwned(c, peer) {
		return
	}
	var iface models.WGInterface
	if err := database.DB.First(&iface, peer.InterfaceID).Error; err != nil {
		respondInterfaceLookupError(c, err)
		return
	}
	cfg := wg.BuildClientConfig(&iface, peer)
	png, err := qrcode.Encode(cfg, qrcode.Medium, 320)
	if err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to render QR")
		return
	}
	c.Data(http.StatusOK, "image/png", png)
}

// --- helpers ---

func findPeer(c *gin.Context) (*models.Peer, error) {
	return findPeerWithScope(c, database.DB)
}

func findPeerUnscoped(c *gin.Context) (*models.Peer, error) {
	return findPeerWithScope(c, database.DB.Unscoped())
}

func findPeerWithScope(c *gin.Context, db *gorm.DB) (*models.Peer, error) {
	idStr := c.Param("peerId")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		dto.Error(c, http.StatusBadRequest, "invalid peer id")
		return nil, errors.New("invalid id")
	}
	var peer models.Peer
	if err := db.First(&peer, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "peer not found")
		} else {
			dto.Error(c, http.StatusInternalServerError, "failed to fetch peer")
		}
		return nil, err
	}
	return &peer, nil
}

// respondInterfaceLookupError answers 404 for a missing interface and 500 for a
// genuine database failure, instead of mislabelling every error as not-found.
func respondInterfaceLookupError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		dto.Error(c, http.StatusNotFound, "interface not found")
		return
	}
	dto.Error(c, http.StatusInternalServerError, "failed to fetch interface")
}

// ensurePeerUnique returns a human-readable conflict message when the peer's
// public key or assigned IP is already taken (a 409), or a non-nil error for a
// genuine DB failure (a 500). An empty message with nil error means it is unique.
// This keeps a real query failure from masquerading as a duplicate conflict.
func ensurePeerUnique(peer models.Peer) (string, error) {
	var count int64
	if err := database.DB.Unscoped().Model(&models.Peer{}).
		Where("public_key = ?", peer.PublicKey).
		Count(&count).Error; err != nil {
		return "", err
	}
	if count > 0 {
		return "peer public key already exists", nil
	}

	count = 0
	if err := database.DB.Unscoped().Model(&models.Peer{}).
		Where("interface_id = ? AND assigned_ip = ?", peer.InterfaceID, peer.AssignedIP).
		Count(&count).Error; err != nil {
		return "", err
	}
	if count > 0 {
		return "assigned IP already exists on this interface", nil
	}
	return "", nil
}

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
