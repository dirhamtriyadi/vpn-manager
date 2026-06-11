package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/example/wg-panel/config"
	"github.com/example/wg-panel/database"
	"github.com/example/wg-panel/dto"
	"github.com/example/wg-panel/middleware"
	"github.com/example/wg-panel/models"
	"github.com/example/wg-panel/wg"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InterfaceHandler groups interface endpoints and holds app config.
type InterfaceHandler struct {
	Cfg *config.Config
}

// NewInterfaceHandler builds an InterfaceHandler.
func NewInterfaceHandler(cfg *config.Config) *InterfaceHandler {
	return &InterfaceHandler{Cfg: cfg}
}

// List godoc
// @Summary      List interfaces
// @Tags         interfaces
// @Produce      json
// @Success      200  {object}  dto.APIResponse
// @Router       /interfaces [get]
func (h *InterfaceHandler) List(c *gin.Context) {
	allowedSorts := map[string]string{
		"id":          "id",
		"name":        "name",
		"listen_port": "listen_port",
		"address":     "address",
		"endpoint":    "endpoint",
		"enabled":     "enabled",
		"created_at":  "created_at",
		"updated_at":  "updated_at",
	}
	list := dto.ParseListQuery(c, allowedSorts, "id")
	query := database.DB.Model(&models.WGInterface{})
	if list.Search != "" {
		like := "%" + list.Search + "%"
		query = query.Where("name LIKE ? OR address LIKE ? OR endpoint LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to count interfaces")
		return
	}

	var ifaces []models.WGInterface
	if err := query.Order(list.OrderClause(allowedSorts)).Limit(list.PerPage).Offset(list.Offset).Find(&ifaces).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch interfaces")
		return
	}
	dto.Paginated(c, "data fetched successfully", ifaces, dto.NewListMeta(list, total))
}

// Get godoc
// @Summary      Get an interface (with peers)
// @Tags         interfaces
// @Produce      json
// @Param        id   path      int  true  "Interface ID"
// @Success      200  {object}  dto.APIResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /interfaces/{id} [get]
func (h *InterfaceHandler) Get(c *gin.Context) {
	iface, err := findInterface(c)
	if err != nil {
		return
	}
	dto.OK(c, "data fetched successfully", iface)
}

// Create godoc
// @Summary      Create an interface
// @Tags         interfaces
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateInterfaceRequest  true  "Interface payload"
// @Success      201   {object}  dto.APIResponse
// @Failure      422   {object}  dto.ErrorResponse
// @Router       /interfaces [post]
func (h *InterfaceHandler) Create(c *gin.Context) {
	var req dto.CreateInterfaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}

	privateKey := req.PrivateKey
	if privateKey == "" {
		kp, err := wg.GenerateKeyPair()
		if err != nil {
			dto.Error(c, http.StatusInternalServerError, "failed to generate keys")
			return
		}
		privateKey = kp.PrivateKey
	}
	publicKey, err := wg.PublicKeyFromPrivate(privateKey)
	if err != nil {
		dto.Error(c, http.StatusUnprocessableEntity, "invalid private key")
		return
	}

	endpoint := req.Endpoint
	if endpoint == "" {
		endpoint = h.Cfg.DefaultEndpoint
	}
	mtu := req.MTU
	if mtu == 0 {
		mtu = 1420
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	iface := models.WGInterface{
		Name:       req.Name,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		ListenPort: req.ListenPort,
		Address:    req.Address,
		DNS:        req.DNS,
		MTU:        mtu,
		Endpoint:   endpoint,
		Enabled:    enabled,
	}

	// GORM soft deletes keep old rows in the table. Because interface name has a
	// unique index, a trashed interface with the same name still blocks creating a
	// duplicate. Ask the user to restore or permanently delete it instead of
	// silently purging trash.
	var deletedIface models.WGInterface
	if err := database.DB.Unscoped().Where("name = ? AND deleted_at IS NOT NULL", req.Name).First(&deletedIface).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		dto.Error(c, http.StatusInternalServerError, "failed to check deleted interfaces")
		return
	} else if err == nil {
		dto.Error(c, http.StatusConflict, "interface name exists in trash; restore or permanently delete it first")
		return
	}

	if err := database.DB.Create(&iface).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to create interface (name/port may already exist)")
		return
	}

	msg := "interface created"
	if err := reconcile(iface.ID); err != nil {
		msg = "interface saved but not applied to kernel: " + err.Error()
	}
	dto.Created(c, msg, iface)
}

// Update godoc
// @Summary      Update an interface
// @Tags         interfaces
// @Accept       json
// @Produce      json
// @Param        id    path      int                          true  "Interface ID"
// @Param        body  body      dto.UpdateInterfaceRequest   true  "Interface payload"
// @Success      200   {object}  dto.APIResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Router       /interfaces/{id} [put]
func (h *InterfaceHandler) Update(c *gin.Context) {
	iface, err := findInterface(c)
	if err != nil {
		return
	}

	var req dto.UpdateInterfaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}

	iface.ListenPort = req.ListenPort
	iface.Address = req.Address
	iface.Endpoint = req.Endpoint
	iface.DNS = req.DNS
	if req.MTU > 0 {
		iface.MTU = req.MTU
	}
	if req.Enabled != nil {
		iface.Enabled = *req.Enabled
	}

	if err := database.DB.Save(iface).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to update interface")
		return
	}

	msg := "interface updated"
	if err := reconcile(iface.ID); err != nil {
		msg = "interface saved but not applied to kernel: " + err.Error()
	}
	dto.OK(c, msg, iface)
}

// Delete godoc
// @Summary      Delete an interface
// @Tags         interfaces
// @Produce      json
// @Param        id   path      int  true  "Interface ID"
// @Success      200  {object}  dto.APIResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /interfaces/{id} [delete]
func (h *InterfaceHandler) Delete(c *gin.Context) {
	iface, err := findInterface(c)
	if err != nil {
		return
	}

	removeErr := wg.RemoveLink(iface.Name)
	if err := database.DB.Where("interface_id = ?", iface.ID).Delete(&models.Peer{}).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to move interface peers to trash")
		return
	}
	if err := database.DB.Delete(iface).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to move interface to trash")
		return
	}

	msg := "interface moved to trash"
	if removeErr != nil {
		msg = "interface moved to trash, but device cleanup failed: " + removeErr.Error()
	}
	dto.NoData(c, http.StatusOK, msg)
}

func (h *InterfaceHandler) Trash(c *gin.Context) {
	allowedSorts := map[string]string{
		"id":         "id",
		"name":       "name",
		"address":    "address",
		"endpoint":   "endpoint",
		"created_at": "created_at",
		"updated_at": "updated_at",
		"deleted_at": "deleted_at",
	}
	list := dto.ParseListQuery(c, allowedSorts, "deleted_at")
	if c.Query("sort_order") == "" {
		list.SortOrder = "desc"
	}
	query := database.DB.Unscoped().Model(&models.WGInterface{}).Where("deleted_at IS NOT NULL")
	if list.Search != "" {
		like := "%" + list.Search + "%"
		query = query.Where("name LIKE ? OR address LIKE ? OR endpoint LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to count trashed interfaces")
		return
	}

	var ifaces []models.WGInterface
	if err := query.Order(list.OrderClause(allowedSorts)).Limit(list.PerPage).Offset(list.Offset).Find(&ifaces).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch trashed interfaces")
		return
	}
	dto.Paginated(c, "data fetched successfully", ifaces, dto.NewListMeta(list, total))
}

func (h *InterfaceHandler) Restore(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var iface models.WGInterface
	if err := database.DB.Unscoped().First(&iface, id).Error; err != nil {
		dto.Error(c, http.StatusNotFound, "interface not found")
		return
	}
	if !iface.DeletedAt.Valid {
		dto.OK(c, "interface already active", iface)
		return
	}

	if err := database.DB.Unscoped().Model(&iface).Update("deleted_at", nil).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to restore interface")
		return
	}
	// Restore peers that were trashed with this interface. Manually deleted peers
	// can still be permanently deleted from Trash if needed.
	if err := database.DB.Unscoped().Model(&models.Peer{}).Where("interface_id = ? AND deleted_at IS NOT NULL", iface.ID).Update("deleted_at", nil).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "interface restored but peers were not restored")
		return
	}
	if err := reconcile(iface.ID); err != nil {
		dto.OK(c, "interface restored but not applied to kernel: "+err.Error(), iface)
		return
	}
	dto.OK(c, "interface restored", iface)
}

func (h *InterfaceHandler) Purge(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var iface models.WGInterface
	if err := database.DB.Unscoped().First(&iface, id).Error; err != nil {
		dto.Error(c, http.StatusNotFound, "interface not found")
		return
	}
	removeErr := wg.RemoveLink(iface.Name)
	if err := database.DB.Unscoped().Where("interface_id = ?", iface.ID).Delete(&models.Peer{}).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to permanently delete interface peers")
		return
	}
	if err := database.DB.Unscoped().Delete(&iface).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to permanently delete interface")
		return
	}
	msg := "interface permanently deleted"
	if removeErr != nil {
		msg = "interface permanently deleted, but device cleanup failed: " + removeErr.Error()
	}
	dto.NoData(c, http.StatusOK, msg)
}

// Sync godoc
// @Summary      Apply interface to the kernel
// @Description  (Re)create the link and push the current peer set to WireGuard
// @Tags         interfaces
// @Produce      json
// @Param        id   path      int  true  "Interface ID"
// @Success      200  {object}  dto.APIResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /interfaces/{id}/sync [post]
func (h *InterfaceHandler) Sync(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := reconcile(id); err != nil {
		dto.Error(c, http.StatusInternalServerError, "sync failed: "+err.Error())
		return
	}
	dto.NoData(c, http.StatusOK, "interface applied to kernel")
}

// Status godoc
// @Summary      Interface runtime status
// @Description  Returns peers enriched with live handshake/transfer stats
// @Tags         interfaces
// @Produce      json
// @Param        id   path      int  true  "Interface ID"
// @Success      200  {object}  dto.APIResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /interfaces/{id}/status [get]
func (h *InterfaceHandler) Status(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var iface models.WGInterface
	if err := database.DB.First(&iface, id).Error; err != nil {
		dto.Error(c, http.StatusNotFound, "interface not found")
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
	query := database.DB.Model(&models.Peer{}).Where("interface_id = ?", id)
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
	iface.Peers = peers

	stats, statErr := wg.DeviceStats(iface.Name)
	now := time.Now()
	for i := range iface.Peers {
		p := &iface.Peers[i]
		if st, ok := stats[p.PublicKey]; ok {
			if !st.LastHandshake.IsZero() {
				p.LastHandshake = st.LastHandshake.Format(time.RFC3339)
				p.Online = now.Sub(st.LastHandshake) < 3*time.Minute
			}
			p.RxBytes = st.RxBytes
			p.TxBytes = st.TxBytes
		}
	}

	resp := gin.H{
		"interface": iface,
		"kernel_up": statErr == nil,
	}
	if statErr != nil {
		resp["kernel_message"] = statErr.Error()
	}
	dto.Paginated(c, "data fetched successfully", resp, dto.NewListMeta(list, total))
}

// --- helpers ---

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		dto.Error(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return uint(id), true
}

func findInterface(c *gin.Context) (*models.WGInterface, error) {
	id, ok := parseID(c)
	if !ok {
		return nil, errors.New("invalid id")
	}
	var iface models.WGInterface
	if err := database.DB.Preload("Peers").First(&iface, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "interface not found")
		} else {
			dto.Error(c, http.StatusInternalServerError, "failed to fetch interface")
		}
		return nil, err
	}
	return &iface, nil
}
