package handlers

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/example/wg-panel/database"
	"github.com/example/wg-panel/dto"
	"github.com/example/wg-panel/middleware"
	"github.com/example/wg-panel/models"
	"github.com/example/wg-panel/openvpn"
	"github.com/example/wg-panel/secrets"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OpenVPNHandler struct{}

func NewOpenVPNHandler() *OpenVPNHandler {
	return &OpenVPNHandler{}
}

func (h *OpenVPNHandler) Roadmap(c *gin.Context) {
	dto.OK(c, "data fetched successfully", dto.OpenVPNRoadmapResponse{
		Available:           false,
		Status:              "roadmap",
		RuntimeMode:         "container_openvpn_preview",
		SecretStorageStatus: "encrypted_secret_scaffold",
		ManifestStatus:      "persisted_manifest_scaffold",
		NextSteps: []string{
			"add container lifecycle management and status parser",
			"add firewall/NAT ownership model",
		},
		BlockedMessage: "OpenVPN is scaffolded but not enabled until runtime lifecycle, status parsing, and firewall ownership are implemented.",
	})
}

func (h *OpenVPNHandler) ListInstances(c *gin.Context) {
	allowedSorts := map[string]string{
		"id":          "id",
		"name":        "name",
		"remote_host": "remote_host",
		"listen_port": "listen_port",
		"protocol":    "protocol",
		"created_at":  "created_at",
		"updated_at":  "updated_at",
	}
	list := dto.ParseListQuery(c, allowedSorts, "id")
	query := database.DB.Model(&models.OpenVPNInstance{})
	if list.Search != "" {
		like := "%" + list.Search + "%"
		query = query.Where("name LIKE ? OR remote_host LIKE ? OR tunnel_cidr LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to count OpenVPN instances")
		return
	}

	var instances []models.OpenVPNInstance
	if err := query.Order(list.OrderClause(allowedSorts)).Limit(list.PerPage).Offset(list.Offset).Find(&instances).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to fetch OpenVPN instances")
		return
	}
	responses := make([]dto.OpenVPNInstanceDraftResponse, 0, len(instances))
	for _, instance := range instances {
		responses = append(responses, openVPNInstanceResponse(instance))
	}
	dto.Paginated(c, "data fetched successfully", responses, dto.NewListMeta(list, total))
}

func (h *OpenVPNHandler) CreateInstanceDraft(c *gin.Context) {
	var req dto.CreateOpenVPNInstanceDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}

	masterKey := strings.TrimSpace(os.Getenv("OPENVPN_SECRET_MASTER_KEY"))
	if masterKey == "" {
		dto.Error(c, http.StatusServiceUnavailable, "OPENVPN_SECRET_MASTER_KEY is required before saving OpenVPN certificate material")
		return
	}
	envelope, err := secrets.NewEnvelope(masterKey)
	if err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to initialize OpenVPN secret envelope")
		return
	}

	draft, err := openvpn.BuildInstanceDraft(openvpn.InstanceDraftInput{
		Name:          req.Name,
		RemoteHost:    req.RemoteHost,
		ListenPort:    req.ListenPort,
		Protocol:      req.Protocol,
		TunnelCIDR:    req.TunnelCIDR,
		DNS:           req.DNS,
		CACertPEM:     req.CACertPEM,
		ServerCertPEM: req.ServerCertPEM,
		ServerKeyPEM:  req.ServerKeyPEM,
		TLSCryptPEM:   req.TLSCryptPEM,
	}, envelope)
	if err != nil {
		dto.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	var deleted models.OpenVPNInstance
	if err := database.DB.Unscoped().Where("name = ? AND deleted_at IS NOT NULL", draft.Instance.Name).First(&deleted).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		dto.Error(c, http.StatusInternalServerError, "failed to check deleted OpenVPN instances")
		return
	} else if err == nil {
		dto.Error(c, http.StatusConflict, "OpenVPN instance name exists in trash; restore or permanently delete it first")
		return
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	if err := tx.Create(&draft.Instance).Error; err != nil {
		tx.Rollback()
		dto.Error(c, http.StatusInternalServerError, "failed to create OpenVPN instance draft")
		return
	}
	for i := range draft.Secrets {
		draft.Secrets[i].OwnerID = draft.Instance.ID
		draft.Secrets[i].Ref = secrets.BuildRef("openvpn-"+draft.Instance.Name, draft.Instance.ID, draft.Secrets[i].Name)
		switch draft.Secrets[i].Name {
		case "ca-cert-pem":
			draft.Instance.CARef = draft.Secrets[i].Ref
		case "server-cert-pem":
			draft.Instance.ServerCertRef = draft.Secrets[i].Ref
		case "server-key-pem":
			draft.Instance.ServerKeyRef = draft.Secrets[i].Ref
		case "tls-crypt-pem":
			draft.Instance.TLSCryptRef = draft.Secrets[i].Ref
		}
		if err := tx.Create(&draft.Secrets[i]).Error; err != nil {
			tx.Rollback()
			dto.Error(c, http.StatusInternalServerError, "failed to store encrypted OpenVPN secret")
			return
		}
	}
	if err := tx.Save(&draft.Instance).Error; err != nil {
		tx.Rollback()
		dto.Error(c, http.StatusInternalServerError, "failed to update OpenVPN secret references")
		return
	}
	if err := tx.Commit().Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to commit OpenVPN instance draft")
		return
	}

	dto.Created(c, "OpenVPN instance draft saved; runtime remains disabled", openVPNInstanceResponse(draft.Instance))
}

func (h *OpenVPNHandler) GetRuntimeManifest(c *gin.Context) {
	instance, ok := findOpenVPNInstanceByParam(c)
	if !ok {
		return
	}
	var manifest models.OpenVPNRuntimeManifest
	if err := database.DB.Where("instance_id = ?", instance.ID).First(&manifest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "OpenVPN runtime manifest has not been generated for this instance")
			return
		}
		dto.Error(c, http.StatusInternalServerError, "failed to fetch OpenVPN runtime manifest")
		return
	}
	dto.OK(c, "data fetched successfully", openVPNManifestResponse(manifest))
}

func (h *OpenVPNHandler) GenerateRuntimeManifest(c *gin.Context) {
	instance, ok := findOpenVPNInstanceByParam(c)
	if !ok {
		return
	}
	record, err := openvpn.BuildRuntimeManifestRecord(instance)
	if err != nil {
		dto.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	var existing models.OpenVPNRuntimeManifest
	err = database.DB.Where("instance_id = ?", instance.ID).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		dto.Error(c, http.StatusInternalServerError, "failed to check existing OpenVPN runtime manifest")
		return
	}
	if err == nil {
		existing.RuntimeMode = record.RuntimeMode
		existing.ServerConf = record.ServerConf
		existing.ComposeYAML = record.ComposeYAML
		existing.Warnings = record.Warnings
		existing.GenerationStatus = record.GenerationStatus
		if err := database.DB.Save(&existing).Error; err != nil {
			dto.Error(c, http.StatusInternalServerError, "failed to update OpenVPN runtime manifest")
			return
		}
		dto.OK(c, "OpenVPN runtime manifest regenerated; runtime remains disabled", openVPNManifestResponse(existing))
		return
	}

	if err := database.DB.Create(&record).Error; err != nil {
		dto.Error(c, http.StatusInternalServerError, "failed to persist OpenVPN runtime manifest")
		return
	}
	dto.Created(c, "OpenVPN runtime manifest generated; runtime remains disabled", openVPNManifestResponse(record))
}

func (h *OpenVPNHandler) PreviewClientProfile(c *gin.Context) {
	var req dto.OpenVPNClientProfilePreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}

	profile, err := openvpn.BuildClientProfile(openvpn.ClientProfileInput{
		ClientName:    req.ClientName,
		RemoteHost:    req.RemoteHost,
		RemotePort:    req.RemotePort,
		Protocol:      req.Protocol,
		CACertPEM:     req.CACertPEM,
		ClientCertPEM: req.ClientCertPEM,
		ClientKeyPEM:  req.ClientKeyPEM,
		TLSAuthPEM:    req.TLSAuthPEM,
	})
	if err != nil {
		dto.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	filename := strings.TrimSpace(req.ClientName)
	if filename == "" {
		filename = "client"
	}
	dto.OK(c, "profile generated", dto.OpenVPNClientProfilePreviewResponse{
		Filename: filename + ".ovpn",
		Profile:  profile,
	})
}

func (h *OpenVPNHandler) PreviewRuntimeManifest(c *gin.Context) {
	var req dto.OpenVPNRuntimeManifestPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := middleware.Validate(req); errs != nil {
		dto.ValidationError(c, errs)
		return
	}

	manifest, err := openvpn.BuildContainerRuntimeManifest(openvpn.RuntimeManifestInput{
		InstanceName: req.InstanceName,
		RemoteHost:   req.RemoteHost,
		ListenPort:   req.ListenPort,
		Protocol:     req.Protocol,
		TunnelCIDR:   req.TunnelCIDR,
		DNS:          req.DNS,
	})
	if err != nil {
		dto.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	dto.OK(c, "runtime manifest generated", dto.OpenVPNRuntimeManifestPreviewResponse{
		RuntimeMode: manifest.RuntimeMode,
		Files:       manifest.Files,
		Warnings:    manifest.Warnings,
	})
}

func findOpenVPNInstanceByParam(c *gin.Context) (models.OpenVPNInstance, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		dto.Error(c, http.StatusBadRequest, "invalid OpenVPN instance id")
		return models.OpenVPNInstance{}, false
	}
	var instance models.OpenVPNInstance
	if err := database.DB.First(&instance, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dto.Error(c, http.StatusNotFound, "OpenVPN instance not found")
			return models.OpenVPNInstance{}, false
		}
		dto.Error(c, http.StatusInternalServerError, "failed to fetch OpenVPN instance")
		return models.OpenVPNInstance{}, false
	}
	return instance, true
}

func openVPNManifestResponse(manifest models.OpenVPNRuntimeManifest) dto.OpenVPNRuntimeManifestResponse {
	warnings := []string{}
	if strings.TrimSpace(manifest.Warnings) != "" {
		warnings = strings.Split(manifest.Warnings, "\n")
	}
	return dto.OpenVPNRuntimeManifestResponse{
		ID:               manifest.ID,
		InstanceID:       manifest.InstanceID,
		RuntimeMode:      manifest.RuntimeMode,
		ServerConf:       manifest.ServerConf,
		ComposeYAML:      manifest.ComposeYAML,
		Warnings:         warnings,
		GenerationStatus: manifest.GenerationStatus,
	}
}

func openVPNInstanceResponse(instance models.OpenVPNInstance) dto.OpenVPNInstanceDraftResponse {
	refs := map[string]string{}
	if instance.CARef != "" {
		refs["ca_cert"] = instance.CARef
	}
	if instance.ServerCertRef != "" {
		refs["server_cert"] = instance.ServerCertRef
	}
	if instance.TLSCryptRef != "" {
		refs["tls_crypt"] = instance.TLSCryptRef
	}
	if instance.ServerKeyRef != "" {
		refs["server_key"] = "stored"
	}
	return dto.OpenVPNInstanceDraftResponse{
		ID:                  instance.ID,
		Name:                instance.Name,
		RemoteHost:          instance.RemoteHost,
		ListenPort:          instance.ListenPort,
		Protocol:            instance.Protocol,
		TunnelCIDR:          instance.TunnelCIDR,
		DNS:                 instance.DNS,
		Enabled:             instance.Enabled,
		RuntimeMode:         instance.RuntimeMode,
		SecretStorageStatus: "encrypted_secret_scaffold",
		SecretRefs:          refs,
	}
}
