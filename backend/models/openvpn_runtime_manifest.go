package models

import (
	"time"

	"gorm.io/gorm"
)

// OpenVPNRuntimeManifest stores generated container runtime files for an OpenVPN
// draft. It does not represent a running service and should not enable OpenVPN.
type OpenVPNRuntimeManifest struct {
	ID               uint   `json:"id" gorm:"primaryKey"`
	InstanceID       uint   `json:"instance_id" gorm:"uniqueIndex;not null"`
	RuntimeMode      string `json:"runtime_mode" gorm:"size:64;not null"`
	ServerConf       string `json:"server_conf" gorm:"type:text;not null"`
	ComposeYAML      string `json:"compose_yaml" gorm:"type:text;not null"`
	Warnings         string `json:"warnings" gorm:"type:text"`
	GenerationStatus string `json:"generation_status" gorm:"size:32;not null;default:generated"`

	Instance OpenVPNInstance `json:"instance,omitempty" gorm:"foreignKey:InstanceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}
