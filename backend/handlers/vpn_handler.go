package handlers

import (
	"net/http"
	"time"

	"github.com/example/wg-panel/database"
	"github.com/example/wg-panel/dto"
	"github.com/example/wg-panel/models"
	vpnsvc "github.com/example/wg-panel/vpn"
	"github.com/example/wg-panel/wg"
	"github.com/gin-gonic/gin"
)

type VPNHandler struct {
	registry *vpnsvc.Registry
}

func NewVPNHandler() *VPNHandler {
	registry, err := vpnsvc.NewDefaultRegistry()
	if err != nil {
		registry = vpnsvc.NewRegistry()
	}
	return &VPNHandler{registry: registry}
}

func (h *VPNHandler) Protocols(c *gin.Context) {
	protocols := []dto.VPNProtocolResponse{
		h.protocolResponse(models.ProtocolWireGuard, "WireGuard"),
		h.protocolResponse(models.ProtocolOpenVPN, "OpenVPN"),
		h.protocolResponse(models.ProtocolL2TPIPsec, "L2TP/IPsec"),
		h.protocolResponse(models.ProtocolSSTP, "SSTP"),
		h.protocolResponse(models.ProtocolPPTP, "PPTP"),
	}
	dto.OK(c, "data fetched successfully", protocols)
}

func (h *VPNHandler) protocolResponse(protocol models.VPNProtocol, label string) dto.VPNProtocolResponse {
	response := dto.VPNProtocolResponse{
		ID:             protocol,
		Label:          label,
		Available:      false,
		LegacyInsecure: protocol.IsLegacyInsecure(),
	}
	if driver, ok := h.registry.Get(protocol); ok {
		capabilities := driver.Capabilities()
		response.Available = true
		response.RuntimeStrategy = capabilities.RuntimeStrategy
		response.ConfigDownload = capabilities.ConfigDownload
		response.QRCode = capabilities.QRCode
		response.RequiresCertificates = capabilities.RequiresCertificates
	}
	return response
}

func (h *VPNHandler) Instances(c *gin.Context) {
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
		dto.Error(c, http.StatusInternalServerError, "failed to count vpn instances")
		return
	}

	var ifaces []models.WGInterface
	if err := query.Order(list.OrderClause(allowedSorts)).Limit(list.PerPage).Offset(list.Offset).Find(&ifaces).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch vpn instances")
		return
	}
	dto.Paginated(c, "data fetched successfully", vpnsvc.MapWGInterfacesToVPNInstances(ifaces), dto.NewListMeta(list, total))
}

func (h *VPNHandler) Instance(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var iface models.WGInterface
	if err := database.DB.First(&iface, id).Error; err != nil {
		dto.Error(c, http.StatusNotFound, "vpn instance not found")
		return
	}
	dto.OK(c, "data fetched successfully", vpnsvc.MapWGInterfaceToVPNInstance(iface))
}

func (h *VPNHandler) Users(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var iface models.WGInterface
	if err := database.DB.First(&iface, id).Error; err != nil {
		dto.Error(c, http.StatusNotFound, "vpn instance not found")
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
		dto.Error(c, http.StatusInternalServerError, "failed to count vpn users")
		return
	}
	var peers []models.Peer
	if err := query.Order(list.OrderClause(allowedSorts)).Limit(list.PerPage).Offset(list.Offset).Find(&peers).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch vpn users")
		return
	}
	dto.Paginated(c, "data fetched successfully", vpnsvc.MapPeersToVPNUsers(peers), dto.NewListMeta(list, total))
}

func (h *VPNHandler) Status(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var iface models.WGInterface
	if err := database.DB.First(&iface, id).Error; err != nil {
		dto.Error(c, http.StatusNotFound, "vpn instance not found")
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
		dto.Error(c, http.StatusInternalServerError, "failed to count vpn users")
		return
	}
	var peers []models.Peer
	if err := query.Order(list.OrderClause(allowedSorts)).Limit(list.PerPage).Offset(list.Offset).Find(&peers).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch vpn users")
		return
	}

	stats, statErr := wg.DeviceStats(iface.Name)
	now := time.Now()
	for i := range peers {
		p := &peers[i]
		if st, ok := stats[p.PublicKey]; ok {
			if !st.LastHandshake.IsZero() {
				p.LastHandshake = st.LastHandshake.Format(time.RFC3339)
				p.Online = now.Sub(st.LastHandshake) < 3*time.Minute
			}
			p.RxBytes = st.RxBytes
			p.TxBytes = st.TxBytes
		}
	}

	instance := vpnsvc.MapWGInterfaceToVPNInstance(iface)
	if statErr == nil {
		instance.Status = "up"
	} else {
		instance.Status = "down"
	}
	resp := gin.H{
		"instance":  instance,
		"users":     vpnsvc.MapPeersToVPNUsers(peers),
		"kernel_up": statErr == nil,
	}
	if statErr != nil {
		resp["kernel_message"] = statErr.Error()
	}
	dto.Paginated(c, "data fetched successfully", resp, dto.NewListMeta(list, total))
}
