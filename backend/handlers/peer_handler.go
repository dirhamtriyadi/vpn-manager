package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/example/wg-panel/database"
	"github.com/example/wg-panel/dto"
	"github.com/example/wg-panel/middleware"
	"github.com/example/wg-panel/models"
	"github.com/example/wg-panel/wg"
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

	if err := database.DB.Create(&peer).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to create peer (public key or IP may already exist)")
		return
	}

	msg := "peer created"
	if err := syncPeer(iface.ID, peer, false); err != nil {
		msg = "peer saved but not applied to kernel: " + err.Error()
	}
	dto.Created(c, msg, peer)
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
	peer.PersistentKeepalive = req.PersistentKeepalive
	if req.Enabled != nil {
		peer.Enabled = *req.Enabled
	}

	if err := database.DB.Save(peer).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to update peer")
		return
	}

	msg := "peer updated"
	if err := syncPeer(peer.InterfaceID, *peer, false); err != nil {
		msg = "peer saved but not applied to kernel: " + err.Error()
	}
	dto.OK(c, msg, peer)
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
	ifaceID := peer.InterfaceID
	peerSnapshot := *peer
	if err := database.DB.Delete(peer).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to move peer to trash")
		return
	}
	msg := "peer moved to trash"
	if err := syncPeer(ifaceID, peerSnapshot, true); err != nil {
		msg = "peer moved to trash but kernel not updated: " + err.Error()
	}
	dto.NoData(c, http.StatusOK, msg)
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
	query := database.DB.Unscoped().Model(&models.Peer{}).Where("deleted_at IS NOT NULL")
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
	if !peer.DeletedAt.Valid {
		dto.OK(c, "peer already active", peer)
		return
	}
	var iface models.WGInterface
	if err := database.DB.First(&iface, peer.InterfaceID).Error; err != nil {
		dto.Error(c, http.StatusConflict, "restore the parent interface before restoring this peer")
		return
	}
	if err := database.DB.Unscoped().Model(peer).Update("deleted_at", nil).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to restore peer")
		return
	}
	msg := "peer restored"
	if err := syncPeer(peer.InterfaceID, *peer, false); err != nil {
		msg = "peer restored but kernel not updated: " + err.Error()
	}
	dto.OK(c, msg, peer)
}

func (h *PeerHandler) Purge(c *gin.Context) {
	peer, err := findPeerUnscoped(c)
	if err != nil {
		return
	}
	peerSnapshot := *peer
	if err := database.DB.Unscoped().Delete(peer).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to permanently delete peer")
		return
	}
	msg := "peer permanently deleted"
	if err := syncPeer(peerSnapshot.InterfaceID, peerSnapshot, true); err != nil {
		msg = "peer permanently deleted but kernel not updated: " + err.Error()
	}
	dto.NoData(c, http.StatusOK, msg)
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
	var iface models.WGInterface
	if err := database.DB.First(&iface, peer.InterfaceID).Error; err != nil {
		dto.Error(c, http.StatusNotFound, "interface not found")
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
	var iface models.WGInterface
	if err := database.DB.First(&iface, peer.InterfaceID).Error; err != nil {
		dto.Error(c, http.StatusNotFound, "interface not found")
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

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
