package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const AlgorithmAES256GCM = "AES-256-GCM"

type SealedSecret struct {
	Algorithm  string `json:"algorithm"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type Envelope struct {
	aead cipher.AEAD
}

func NewEnvelope(masterKey string) (*Envelope, error) {
	if strings.TrimSpace(masterKey) == "" {
		return nil, fmt.Errorf("secret master key is required")
	}
	key := sha256.Sum256([]byte(masterKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Envelope{aead: aead}, nil
}

func (e *Envelope) Encrypt(plaintext string) (SealedSecret, error) {
	if e == nil || e.aead == nil {
		return SealedSecret{}, fmt.Errorf("secret envelope is not initialized")
	}
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return SealedSecret{}, err
	}
	sealed := e.aead.Seal(nil, nonce, []byte(plaintext), nil)
	return SealedSecret{
		Algorithm:  AlgorithmAES256GCM,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(sealed),
	}, nil
}

func (e *Envelope) Decrypt(secret SealedSecret) (string, error) {
	if e == nil || e.aead == nil {
		return "", fmt.Errorf("secret envelope is not initialized")
	}
	if secret.Algorithm != AlgorithmAES256GCM {
		return "", fmt.Errorf("unsupported secret algorithm")
	}
	nonce, err := base64.StdEncoding.DecodeString(secret.Nonce)
	if err != nil {
		return "", fmt.Errorf("invalid secret nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(secret.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("invalid secret ciphertext: %w", err)
	}
	plaintext, err := e.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret")
	}
	return string(plaintext), nil
}

var unsafeRefChars = regexp.MustCompile(`[^a-z0-9-]+`)

func BuildRef(scope string, ownerID uint, name string) string {
	parts := []string{sanitizeRefPart(scope), fmt.Sprintf("%d", ownerID), sanitizeRefPart(name)}
	return strings.Join(parts, "/")
}

func sanitizeRefPart(value string) string {
	clean := strings.ToLower(strings.TrimSpace(value))
	clean = strings.ReplaceAll(clean, "_", "-")
	clean = unsafeRefChars.ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "-")
	return clean
}
