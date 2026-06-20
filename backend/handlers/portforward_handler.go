package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/vpn-manager/config"
	"github.com/example/vpn-manager/database"
	"github.com/example/vpn-manager/dto"
	"github.com/example/vpn-manager/middleware"
	"github.com/example/vpn-manager/models"
	"github.com/example/vpn-manager/wg"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PortForwardHandler manages public-IP port forwards: it exposes a port on the
// server's WAN and DNATs it through the tunnel to a peer's tunnel IP.
type PortForwardHandler struct {
	Cfg *config.Config
}

func NewPortForwardHandler(cfg *config.Config) *PortForwardHandler {
	return &PortForwardHandler{Cfg: cfg}
}

// reservedPort blocks forwarding host-critical ports into a tunnel: doing so
// would DNAT the host's own SSH / management panel / WireGuard traffic to a
// peer (lockout or MITM). DNAT runs in PREROUTING, before local delivery.
func (h *PortForwardHandler) reservedPort(port int) (bool, string) {
	if port == 22 {
		return true, "SSH"
	}
	if h.Cfg != nil {
		if sp, err := strconv.Atoi(strings.TrimSpace(h.Cfg.ServerPort)); err == nil && sp == port {
			return true, "the management panel"
		}
	}
	var count int64
	if err := database.DB.Unscoped().Model(&models.WGInterface{}).Where("listen_port = ?", port).Count(&count).Error; err == nil && count > 0 {
		return true, "a WireGuard interface listen port"
	}
	return false, ""
}

// List godoc
// @Summary      List port forwards
// @Description  Lists the caller's public-IP port forwards, optionally filtered by interface
// @Tags         port-forwards
// @Produce      json
// @Param        interface_id  query     int  false  "Filter by interface ID"
// @Success      200  {object}  dto.APIResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /port-forwards [get]
func (h *PortForwardHandler) List(c *gin.Context) {
	query := scopeOwned(c, database.DB.Model(&models.PortForward{}))
	if ifaceID := strings.TrimSpace(c.Query("interface_id")); ifaceID != "" {
		query = query.Where("interface_id = ?", ifaceID)
	}
	var pfs []models.PortForward
	if err := query.Order("id asc").Find(&pfs).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch port forwards")
		return
	}
	ifaceNames, peerNames := portForwardNames(pfs)
	out := make([]dto.PortForwardResponse, 0, len(pfs))
	for i := range pfs {
		out = append(out, portForwardResponse(pfs[i], ifaceNames[pfs[i].InterfaceID], peerNames[pfs[i].PeerID]))
	}
	dto.OK(c, "data fetched successfully", out)
}

// Get godoc
// @Summary      Get a port forward
// @Tags         port-forwards
// @Produce      json
// @Param        id   path      int  true  "Port forward ID"
// @Success      200  {object}  dto.APIResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /port-forwards/{id} [get]
func (h *PortForwardHandler) Get(c *gin.Context) {
	pf, iface, ok := findOwnedPortForward(c)
	if !ok {
		return
	}
	dto.OK(c, "data fetched successfully", portForwardResponse(*pf, iface.Name, peerNameByID(pf.PeerID)))
}

// Create godoc
// @Summary      Create a port forward
// @Description  Exposes a public port on the server WAN and DNATs it through the tunnel to a peer
// @Tags         port-forwards
// @Accept       json
// @Produce      json
// @Param        request  body      dto.CreatePortForwardRequest  true  "Port forward"
// @Success      201  {object}  dto.APIResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      409  {object}  dto.ErrorResponse
// @Failure      422  {object}  dto.ErrorResponse
// @Router       /port-forwards [post]
func (h *PortForwardHandler) Create(c *gin.Context) {
	var req dto.CreatePortForwardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}
	if reserved, why := h.reservedPort(req.PublicPort); reserved {
		dto.Error(c, http.StatusUnprocessableEntity, fmt.Sprintf("public port %d is reserved by %s; choose a different public port", req.PublicPort, why))
		return
	}

	iface, ok := getOwnedInterface(c, req.InterfaceID)
	if !ok {
		return
	}
	var peer models.Peer
	if err := database.DB.First(&peer, req.PeerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "peer not found")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to fetch peer")
		return
	}
	if peer.InterfaceID != iface.ID {
		dto.Error(c, http.StatusUnprocessableEntity, "peer does not belong to the selected interface")
		return
	}
	if strings.TrimSpace(peer.AssignedIP) == "" {
		dto.Error(c, http.StatusUnprocessableEntity, "peer has no tunnel IP to forward to")
		return
	}

	egress, ok := resolveEgress(c, iface)
	if !ok {
		return
	}

	pf := models.PortForward{
		InterfaceID: iface.ID,
		PeerID:      peer.ID,
		Protocol:    strings.ToLower(strings.TrimSpace(req.Protocol)),
		PublicPort:  req.PublicPort,
		TargetPort:  req.TargetPort,
		TargetIP:    peer.AssignedIP,
		Egress:      egress,
		Enabled:     true,
		Comment:     strings.TrimSpace(req.Comment),
		OwnerID:     currentOwnerID(c),
	}
	if err := database.DB.Create(&pf).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			dto.Error(c, http.StatusConflict, "a port forward for this protocol and public port already exists")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to create port forward")
		return
	}

	resp := portForwardResponse(pf, iface.Name, peer.Name)
	if err := wg.ApplyPortForward(pf, iface.Name); err != nil {
		dto.CreatedWarn(c, "port forward created", resp, "saved but firewall rules not applied: "+err.Error())
		return
	}
	dto.Created(c, "port forward created", resp)
}

// Update godoc
// @Summary      Update a port forward
// @Description  Updates the target port, comment, or enabled state and re-applies firewall rules
// @Tags         port-forwards
// @Accept       json
// @Produce      json
// @Param        id       path      int                           true  "Port forward ID"
// @Param        request  body      dto.UpdatePortForwardRequest  true  "Fields to update"
// @Success      200  {object}  dto.APIResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /port-forwards/{id} [put]
func (h *PortForwardHandler) Update(c *gin.Context) {
	pf, iface, ok := findOwnedPortForward(c)
	if !ok {
		return
	}
	var req dto.UpdatePortForwardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}

	// Remove the current rules first (they key on the current port/IP), then
	// re-apply with the new state.
	_ = wg.RemovePortForward(*pf, iface.Name)

	if req.TargetPort > 0 {
		pf.TargetPort = req.TargetPort
	}
	pf.Comment = strings.TrimSpace(req.Comment)
	if req.Enabled != nil {
		pf.Enabled = *req.Enabled
	}
	if err := database.DB.Save(pf).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to update port forward")
		return
	}

	resp := portForwardResponse(*pf, iface.Name, peerNameByID(pf.PeerID))
	if pf.Enabled {
		if err := wg.ApplyPortForward(*pf, iface.Name); err != nil {
			dto.OKWarn(c, "port forward updated", resp, "saved but firewall rules not applied: "+err.Error())
			return
		}
	}
	dto.OK(c, "port forward updated", resp)
}

// Delete godoc
// @Summary      Delete a port forward
// @Description  Removes the firewall rules and the record permanently
// @Tags         port-forwards
// @Produce      json
// @Param        id   path      int  true  "Port forward ID"
// @Success      200  {object}  dto.APIResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /port-forwards/{id} [delete]
func (h *PortForwardHandler) Delete(c *gin.Context) {
	pf, iface, ok := findOwnedPortForward(c)
	if !ok {
		return
	}
	_ = wg.RemovePortForward(*pf, iface.Name)
	if err := database.DB.Delete(pf).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to delete port forward")
		return
	}
	dto.NoData(c, http.StatusOK, "port forward deleted")
}

// --- helpers ---

// resolveEgress returns the WAN interface to DNAT from: the interface's
// configured egress, else the host's default-route interface.
func resolveEgress(c *gin.Context, iface *models.WGInterface) (string, bool) {
	egress := strings.TrimSpace(iface.EgressInterface)
	if egress != "" {
		return egress, true
	}
	detected, err := wg.DefaultEgressInterface()
	if err != nil {
		dto.Error(c, http.StatusInternalServerError, "could not detect the server WAN/egress interface: "+err.Error())
		return "", false
	}
	return detected, true
}

func findOwnedPortForward(c *gin.Context) (*models.PortForward, *models.WGInterface, bool) {
	id, ok := parseID(c)
	if !ok {
		return nil, nil, false
	}
	var pf models.PortForward
	if err := database.DB.First(&pf, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "port forward not found")
		} else {
			dto.Error(c, http.StatusInternalServerError, "failed to fetch port forward")
		}
		return nil, nil, false
	}
	if !ownsResource(c, pf.OwnerID) {
		dto.Error(c, http.StatusNotFound, "port forward not found")
		return nil, nil, false
	}
	var iface models.WGInterface
	if err := database.DB.Unscoped().First(&iface, pf.InterfaceID).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch the port forward's interface")
		return nil, nil, false
	}
	return &pf, &iface, true
}

func portForwardResponse(pf models.PortForward, ifaceName, peerName string) dto.PortForwardResponse {
	return dto.PortForwardResponse{
		ID:            pf.ID,
		InterfaceID:   pf.InterfaceID,
		InterfaceName: ifaceName,
		PeerID:        pf.PeerID,
		PeerName:      peerName,
		Protocol:      pf.Protocol,
		PublicPort:    pf.PublicPort,
		TargetPort:    pf.TargetPort,
		TargetIP:      pf.TargetIP,
		Egress:        pf.Egress,
		Enabled:       pf.Enabled,
		Comment:       pf.Comment,
		CreatedAt:     pf.CreatedAt,
		UpdatedAt:     pf.UpdatedAt,
	}
}

func portForwardNames(pfs []models.PortForward) (map[uint]string, map[uint]string) {
	ifaceIDs := map[uint]bool{}
	peerIDs := map[uint]bool{}
	for _, pf := range pfs {
		ifaceIDs[pf.InterfaceID] = true
		peerIDs[pf.PeerID] = true
	}
	ifaceNames := map[uint]string{}
	if len(ifaceIDs) > 0 {
		var ifaces []models.WGInterface
		database.DB.Unscoped().Select("id", "name").Where("id IN ?", keysOf(ifaceIDs)).Find(&ifaces)
		for _, i := range ifaces {
			ifaceNames[i.ID] = i.Name
		}
	}
	peerNames := map[uint]string{}
	if len(peerIDs) > 0 {
		var peers []models.Peer
		database.DB.Unscoped().Select("id", "name").Where("id IN ?", keysOf(peerIDs)).Find(&peers)
		for _, p := range peers {
			peerNames[p.ID] = p.Name
		}
	}
	return ifaceNames, peerNames
}

func peerNameByID(id uint) string {
	var peer models.Peer
	if err := database.DB.Unscoped().Select("name").First(&peer, id).Error; err != nil {
		return ""
	}
	return peer.Name
}

func interfaceNameByID(id uint) string {
	var iface models.WGInterface
	if err := database.DB.Unscoped().Select("name").First(&iface, id).Error; err != nil {
		return ""
	}
	return iface.Name
}

func keysOf(m map[uint]bool) []uint {
	out := make([]uint, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// removePortForwardRules removes the iptables rules for matching forwards but
// KEEPS the DB rows. Used on trash (soft delete) so a later restore re-applies.
func removePortForwardRules(wgDevice string, query *gorm.DB) {
	var pfs []models.PortForward
	if err := query.Find(&pfs).Error; err != nil {
		return
	}
	for _, pf := range pfs {
		if err := wg.RemovePortForward(pf, wgDevice); err != nil {
			log.Printf("port-forward %d: rule removal failed: %v", pf.ID, err)
		}
	}
}

// purgePortForwards removes the rules AND the rows. Used on permanent deletion.
func purgePortForwards(wgDevice string, query *gorm.DB) {
	var pfs []models.PortForward
	if err := query.Find(&pfs).Error; err != nil {
		return
	}
	for _, pf := range pfs {
		if err := wg.RemovePortForward(pf, wgDevice); err != nil {
			log.Printf("port-forward %d: rule removal failed during purge: %v", pf.ID, err)
		}
		if err := database.DB.Delete(&models.PortForward{}, pf.ID).Error; err != nil {
			log.Printf("port-forward %d: row delete failed: %v", pf.ID, err)
		}
	}
}

// reapplyPortForwards re-installs rules for matching ENABLED forwards (restore).
func reapplyPortForwards(wgDevice string, query *gorm.DB) {
	var pfs []models.PortForward
	if err := query.Where("enabled = ?", true).Find(&pfs).Error; err != nil {
		return
	}
	for _, pf := range pfs {
		if err := wg.ApplyPortForward(pf, wgDevice); err != nil {
			log.Printf("port-forward %d: re-apply failed: %v", pf.ID, err)
		}
	}
}

// Trash (rules only — rows kept so restore recovers them).
func removePortForwardRulesForInterface(ifaceName string, ifaceID uint) {
	removePortForwardRules(ifaceName, database.DB.Where("interface_id = ?", ifaceID))
}
func removePortForwardRulesForPeer(ifaceName string, peerID uint) {
	removePortForwardRules(ifaceName, database.DB.Where("peer_id = ?", peerID))
}

// Restore / enable (re-apply enabled rules).
func reapplyPortForwardsForInterface(ifaceName string, ifaceID uint) {
	reapplyPortForwards(ifaceName, database.DB.Where("interface_id = ?", ifaceID))
}
func reapplyPortForwardsForPeer(ifaceName string, peerID uint) {
	reapplyPortForwards(ifaceName, database.DB.Where("peer_id = ?", peerID))
}

// Purge (rules + rows).
func purgePortForwardsForInterface(ifaceName string, ifaceID uint) {
	purgePortForwards(ifaceName, database.DB.Where("interface_id = ?", ifaceID))
}
func purgePortForwardsForPeer(ifaceName string, peerID uint) {
	purgePortForwards(ifaceName, database.DB.Where("peer_id = ?", peerID))
}

// ReapplyAllPortForwards re-installs the iptables rules for every enabled port
// forward whose parent interface is enabled and not trashed. Called on startup
// (after interfaces are reconciled) so the rules survive a host reboot.
func ReapplyAllPortForwards() {
	var ifaces []models.WGInterface
	if err := database.DB.Where("enabled = ?", true).Find(&ifaces).Error; err != nil {
		log.Printf("port-forward reapply: failed to load interfaces: %v", err)
		return
	}
	names := map[uint]string{}
	ids := make([]uint, 0, len(ifaces))
	for i := range ifaces {
		names[ifaces[i].ID] = ifaces[i].Name
		ids = append(ids, ifaces[i].ID)
	}
	if len(ids) == 0 {
		return
	}
	var pfs []models.PortForward
	if err := database.DB.Where("enabled = ? AND interface_id IN ?", true, ids).Find(&pfs).Error; err != nil {
		log.Printf("port-forward reapply: failed to load forwards: %v", err)
		return
	}
	applied := 0
	for _, pf := range pfs {
		name := names[pf.InterfaceID]
		if name == "" {
			continue
		}
		if err := wg.ApplyPortForward(pf, name); err != nil {
			log.Printf("port-forward reapply: forward %d not applied: %v", pf.ID, err)
			continue
		}
		applied++
	}
	if len(pfs) > 0 {
		log.Printf("port-forward reapply: %d/%d applied", applied, len(pfs))
	}
}

// BootstrapRuntime restores the desired runtime state on startup: it reconciles
// every enabled WireGuard interface (so devices/NAT exist after a host reboot)
// then re-applies port forwards. Best effort; failures are logged.
func BootstrapRuntime() {
	var ifaces []models.WGInterface
	if err := database.DB.Where("enabled = ?", true).Find(&ifaces).Error; err != nil {
		log.Printf("bootstrap: failed to load interfaces: %v", err)
		return
	}
	for i := range ifaces {
		if err := reconcile(ifaces[i].ID); err != nil {
			log.Printf("bootstrap: interface %s not applied to kernel: %v", ifaces[i].Name, err)
		}
	}
	ReapplyAllPortForwards()
}
