package models

import "time"

// EncryptedSecret stores encrypted certificate/key material references. Ciphertext
// is intentionally omitted from JSON responses; handlers should expose only Ref.
type EncryptedSecret struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	Ref        string `json:"ref" gorm:"size:255;uniqueIndex;not null"`
	Scope      string `json:"scope" gorm:"size:64;index;not null"`
	OwnerType  string `json:"owner_type" gorm:"size:64;index;not null"`
	OwnerID    uint   `json:"owner_id" gorm:"index;not null"`
	Name       string `json:"name" gorm:"size:128;not null"`
	Algorithm  string `json:"algorithm" gorm:"size:64;not null"`
	Nonce      string `json:"-" gorm:"type:text;not null"`
	Ciphertext string `json:"-" gorm:"type:text;not null"`
	KeyVersion string `json:"key_version" gorm:"size:64;not null;default:v1"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
