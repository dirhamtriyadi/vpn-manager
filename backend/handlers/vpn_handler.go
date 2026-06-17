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

type VPNHandler struct{}

func NewVPNHandler() *VPNHandler {
	return &VPNHandler{}
}

func (h *VPNHandler) Protocols(c *gin.Context) {
	protocols := []dto.VPNProtocolResponse{
		{ID: models.ProtocolWireGuard, Label: "WireGuard", Available: true, LegacyInsecure: models.ProtocolWireGuard.IsLegacyInsecure()},
		{ID: models.ProtocolOpenVPN, Label: "OpenVPN", Available: false, LegacyInsecure: models.ProtocolOpenVPN.IsLegacyInsecure()},
		{ID: models.ProtocolL2TPIPsec, Label: "L2TP/IPsec", Available: false, LegacyInsecure: models.ProtocolL2TPIPsec.IsLegacyInsecure()},
		{ID: models.ProtocolSSTP, Label: "SSTP", Available: false, LegacyInsecure: models.ProtocolSSTP.IsLegacyInsecure()},
		{ID: models.ProtocolPPTP, Label: "PPTP", Available: false, LegacyInsecure: models.ProtocolPPTP.IsLegacyInsecure()},
	}
	dto.OK(c, "data fetched successfully", protocols)
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
