package handlers

import (
	"net/http"
	"os"
	"strings"
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
	protocols := make([]dto.VPNProtocolResponse, 0, len(vpnsvc.AllProtocolSpecs()))
	for _, spec := range vpnsvc.AllProtocolSpecs() {
		protocols = append(protocols, h.protocolResponse(spec))
	}
	dto.OK(c, "data fetched successfully", protocols)
}

func (h *VPNHandler) protocolResponse(spec vpnsvc.ProtocolSpec) dto.VPNProtocolResponse {
	capabilities := spec.Capabilities
	available := h.registry.Supports(spec.Protocol)
	if driver, ok := h.registry.Get(spec.Protocol); ok {
		capabilities = driver.Capabilities()
	}
	return dto.VPNProtocolResponse{
		ID:                   spec.Protocol,
		Label:                spec.Label,
		Status:               spec.Status,
		Description:          spec.Description,
		Available:            available,
		LegacyInsecure:       spec.LegacyInsecure,
		RuntimeStrategy:      capabilities.RuntimeStrategy,
		ConfigDownload:       capabilities.ConfigDownload,
		QRCode:               capabilities.QRCode,
		RequiresCertificates: capabilities.RequiresCertificates,
	}
}

func (h *VPNHandler) ProtocolRoadmap(c *gin.Context) {
	protocol := models.VPNProtocol(c.Param("protocol"))
	roadmap, err := vpnsvc.BuildProtocolRoadmap(protocol, envFlagAny("VPN_RUNTIME_EXECUTION_ENABLED"), envFlagAny("VPN_FIREWALL_APPLY_ENABLED"), envFlagAny("VPN_HOST_VERIFICATION_PASSED"))
	if err != nil {
		dto.Error(c, http.StatusNotFound, "vpn protocol roadmap not found")
		return
	}
	if h.registry.Supports(protocol) {
		roadmap.Available = true
		roadmap.Status = vpnsvc.ProtocolStatusAvailable
		roadmap.BlockedMessage = "Protocol has a registered runtime driver."
		roadmap.EnablementBlockers = []string{}
		roadmap.EnablementReady = true
	}
	dto.OK(c, "data fetched successfully", roadmap)
}

func (h *VPNHandler) ProtocolServicePlan(c *gin.Context) {
	protocol := models.VPNProtocol(c.Param("protocol"))
	plan, err := vpnsvc.BuildProtocolServicePlan(protocol)
	if err != nil {
		dto.Error(c, http.StatusNotFound, "vpn protocol service plan not found")
		return
	}
	dto.OK(c, "data fetched successfully", plan)
}

func envFlagAny(key string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
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
