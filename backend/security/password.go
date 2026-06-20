// Package security hashes and verifies passwords with Argon2id.
//
// Hashes are stored in the standard PHC string format so the parameters travel
// with the hash and can evolve without a migration:
//
//	$argon2id$v=19$m=65536,t=3,p=2$<base64 salt>$<base64 key>
package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// argon2Params are the current hashing parameters. They are encoded into every
// hash, so raising them only affects newly created hashes; old hashes still
// verify against their own embedded parameters.
type argon2Params struct {
	memory  uint32 // KiB
	time    uint32 // iterations
	threads uint8
	saltLen uint32
	keyLen  uint32
}

var defaultParams = argon2Params{
	memory:  64 * 1024, // 64 MiB
	time:    3,
	threads: 2,
	saltLen: 16,
	keyLen:  32,
}

// ErrInvalidHash is returned when an encoded hash cannot be parsed.
var ErrInvalidHash = errors.New("invalid argon2 hash")

// ErrIncompatibleVersion is returned when the hash was produced by a newer
// argon2 version than this binary supports.
var ErrIncompatibleVersion = errors.New("incompatible argon2 version")

// HashPassword returns a PHC-encoded Argon2id hash of the password.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password must not be empty")
	}
	p := defaultParams
	salt := make([]byte, p.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.threads, p.keyLen)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.memory, p.time, p.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether password matches the PHC-encoded hash. The
// comparison is constant time. A malformed hash returns an error (never a
// silent false) so misconfiguration is visible.
func VerifyPassword(encoded, password string) (bool, error) {
	p, salt, key, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}
	other := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.threads, p.keyLen)
	if subtle.ConstantTimeEq(int32(len(key)), int32(len(other))) == 0 {
		return false, nil
	}
	return subtle.ConstantTimeCompare(key, other) == 1, nil
}

func decodeHash(encoded string) (argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// "", "argon2id", "v=19", "m=..,t=..,p=..", salt, key
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argon2Params{}, nil, nil, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return argon2Params{}, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return argon2Params{}, nil, nil, ErrIncompatibleVersion
	}
	var p argon2Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return argon2Params{}, nil, nil, ErrInvalidHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2Params{}, nil, nil, ErrInvalidHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2Params{}, nil, nil, ErrInvalidHash
	}
	p.saltLen = uint32(len(salt))
	p.keyLen = uint32(len(key))
	return p, salt, key, nil
}
