package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/example/vpn-manager/database"
	"github.com/example/vpn-manager/dto"
	"github.com/example/vpn-manager/middleware"
	"github.com/example/vpn-manager/models"
	"github.com/example/vpn-manager/runtimeexec"
	vpnsvc "github.com/example/vpn-manager/vpn"
	"github.com/example/vpn-manager/wg"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	roadmap, err := vpnsvc.BuildProtocolRoadmap(protocol, vpnExecutionEnabled())
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

func (h *VPNHandler) ProtocolProductionPlan(c *gin.Context) {
	protocol := models.VPNProtocol(c.Param("protocol"))
	plan, err := vpnsvc.BuildProductionPlan(protocol, vpnExecutionEnabled())
	if err != nil {
		dto.Error(c, http.StatusNotFound, "vpn protocol production plan not found")
		return
	}
	dto.OK(c, "data fetched successfully", plan)
}

func (h *VPNHandler) PreviewProtocolConfig(c *gin.Context) {
	var req dto.ProtocolConfigPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}
	preview, err := vpnsvc.BuildProtocolConfigPreview(vpnsvc.ProtocolConfigInput{Protocol: req.Protocol, Name: req.Name, RemoteHost: req.RemoteHost, ListenPort: req.ListenPort, PoolCIDR: req.PoolCIDR, DNS: req.DNS})
	if err != nil {
		dto.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	dto.OK(c, "data fetched successfully", preview)
}

func (h *VPNHandler) ListLegacyInstances(c *gin.Context) {
	protocol := models.VPNProtocol(c.Param("protocol"))
	if !isLegacyProtocol(protocol) {
		dto.Error(c, http.StatusNotFound, "vpn protocol draft endpoint not found")
		return
	}
	var instances []models.LegacyVPNInstance
	if err := scopeOwned(c, database.DB.Where("protocol = ?", protocol)).Order("id asc").Find(&instances).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch vpn instance drafts")
		return
	}
	responses := make([]dto.LegacyVPNInstanceDraftResponse, 0, len(instances))
	for _, instance := range instances {
		responses = append(responses, legacyVPNInstanceResponse(instance))
	}
	dto.OK(c, "data fetched successfully", responses)
}

func (h *VPNHandler) CreateLegacyInstanceDraft(c *gin.Context) {
	protocol := models.VPNProtocol(c.Param("protocol"))
	if !isLegacyProtocol(protocol) {
		dto.Error(c, http.StatusNotFound, "vpn protocol draft endpoint not found")
		return
	}
	var req dto.LegacyVPNInstanceDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Protocol = protocol
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}
	preview, err := vpnsvc.BuildProtocolConfigPreview(vpnsvc.ProtocolConfigInput{Protocol: req.Protocol, Name: req.Name, RemoteHost: req.RemoteHost, ListenPort: req.ListenPort, PoolCIDR: req.PoolCIDR, DNS: req.DNS})
	if err != nil {
		dto.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	instance := models.LegacyVPNInstance{Protocol: req.Protocol, Name: req.Name, RemoteHost: req.RemoteHost, ListenPort: req.ListenPort, PoolCIDR: req.PoolCIDR, DNS: req.DNS, Enabled: false, RuntimeMode: preview.RuntimeMode, SecretRef: preview.SecretRefs["credentials"], CertRef: preview.SecretRefs["tls_cert"], KeyRef: preview.SecretRefs["tls_key"], OwnerID: currentOwnerID(c)}
	if instance.ListenPort == 0 {
		instance.ListenPort = legacyDefaultPort(protocol)
	}
	if err := database.DB.Create(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			dto.Error(c, http.StatusConflict, "vpn instance name already in use")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to create vpn instance draft")
		return
	}
	dto.Created(c, "vpn instance draft saved; runtime remains disabled until applied", legacyVPNInstanceResponse(instance))
}

func (h *VPNHandler) ApplyLegacyInstance(c *gin.Context) {
	protocol := models.VPNProtocol(c.Param("protocol"))
	if !isLegacyProtocol(protocol) {
		dto.Error(c, http.StatusNotFound, "vpn protocol apply endpoint not found")
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var instance models.LegacyVPNInstance
	if err := database.DB.Where("protocol = ? AND id = ?", protocol, id).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "vpn instance draft not found")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to fetch vpn instance draft")
		return
	}
	if !ownsResource(c, instance.OwnerID) {
		dto.Error(c, http.StatusNotFound, "vpn instance draft not found")
		return
	}
	files, commands, err := vpnsvc.BuildLegacyRuntimeApplyPlan(vpnsvc.ProtocolConfigInput{Protocol: instance.Protocol, Name: instance.Name, RemoteHost: instance.RemoteHost, ListenPort: instance.ListenPort, PoolCIDR: instance.PoolCIDR, DNS: instance.DNS}, instance.ID)
	if err != nil {
		dto.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	result, err := runtimeexec.Apply(c.Request.Context(), runtimeexec.Options{RootDir: runtimeRootDir(), ExecutionEnabled: vpnExecutionEnabled(), AllowAbsolutePaths: true}, runtimeexec.ApplyPlan{Files: files, Commands: commands})
	if err != nil {
		if errors.Is(err, runtimeexec.ErrExecutionDisabled) {
			dto.Error(c, http.StatusPreconditionFailed, err.Error())
			return
		}
		dto.Error(c, http.StatusInternalServerError, "vpn runtime apply failed: "+err.Error())
		return
	}
	instance.Enabled = true
	instance.RuntimeMode = string(instance.Protocol) + "_active"
	if err := database.DB.Save(&instance).Error; err != nil {
		dto.OKWarn(c, "vpn runtime applied", result, "applied but failed to persist instance state: "+err.Error())
		return
	}
	dto.OK(c, "vpn runtime applied", result)
}

func legacyVPNInstanceResponse(instance models.LegacyVPNInstance) dto.LegacyVPNInstanceDraftResponse {
	refs := map[string]string{}
	if instance.SecretRef != "" {
		refs["credentials"] = instance.SecretRef
	}
	if instance.CertRef != "" {
		refs["tls_cert"] = instance.CertRef
	}
	if instance.KeyRef != "" {
		refs["tls_key"] = instance.KeyRef
	}
	return dto.LegacyVPNInstanceDraftResponse{ID: instance.ID, Protocol: instance.Protocol, Name: instance.Name, RemoteHost: instance.RemoteHost, ListenPort: instance.ListenPort, PoolCIDR: instance.PoolCIDR, DNS: instance.DNS, Enabled: instance.Enabled, RuntimeMode: instance.RuntimeMode, SecretStorageStatus: "encrypted_secret_ref_scaffold", SecretRefs: refs}
}

func isLegacyProtocol(protocol models.VPNProtocol) bool {
	return protocol == models.ProtocolL2TPIPsec || protocol == models.ProtocolSSTP || protocol == models.ProtocolPPTP
}

func legacyDefaultPort(protocol models.VPNProtocol) int {
	switch protocol {
	case models.ProtocolL2TPIPsec:
		return 1701
	case models.ProtocolSSTP:
		return 443
	case models.ProtocolPPTP:
		return 1723
	default:
		return 0
	}
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
	query := scopeOwned(c, database.DB.Model(&models.WGInterface{}))
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "vpn instance not found")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to fetch vpn instance")
		return
	}
	if !ownsResource(c, iface.OwnerID) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "vpn instance not found")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to fetch vpn instance")
		return
	}
	if !ownsResource(c, iface.OwnerID) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "vpn instance not found")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to fetch vpn instance")
		return
	}
	if !ownsResource(c, iface.OwnerID) {
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
