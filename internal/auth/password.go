package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Version = argon2.Version
)

// PasswordConfig holds Argon2id parameters.
type PasswordConfig struct {
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
	SaltLen uint32
}

// DefaultPasswordConfig returns sensible production defaults.
func DefaultPasswordConfig() PasswordConfig {
	return PasswordConfig{
		Time:    3,
		Memory:  65536,
		Threads: 4,
		KeyLen:  32,
		SaltLen: 16,
	}
}

// PasswordHasher handles password hashing and verification using Argon2id.
type PasswordHasher struct {
	cfg PasswordConfig
}

// NewPasswordHasher creates a new PasswordHasher with the given config.
func NewPasswordHasher(cfg PasswordConfig) *PasswordHasher {
	return &PasswordHasher{cfg: cfg}
}

// Hash generates an Argon2id hash of the password.
// Format: $argon2id$v={version}$m={memory},t={time},p={threads}${salt}${hash}
func (p *PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, p.cfg.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, p.cfg.Time, p.cfg.Memory, p.cfg.Threads, p.cfg.KeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version, p.cfg.Memory, p.cfg.Time, p.cfg.Threads, b64Salt, b64Hash,
	)

	return encodedHash, nil
}

// Verify compares a password against an encoded Argon2id hash.
func (p *PasswordHasher) Verify(password, encodedHash string) (bool, error) {
	cfg, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}

	otherHash := argon2.IDKey([]byte(password), salt, cfg.Time, cfg.Memory, cfg.Threads, cfg.KeyLen)

	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}

func decodeHash(encodedHash string) (PasswordConfig, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return PasswordConfig{}, nil, nil, fmt.Errorf("invalid hash format")
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return PasswordConfig{}, nil, nil, fmt.Errorf("invalid version: %w", err)
	}
	if version != argon2Version {
		return PasswordConfig{}, nil, nil, fmt.Errorf("incompatible argon2 version")
	}

	var cfg PasswordConfig
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &cfg.Memory, &cfg.Time, &cfg.Threads)
	if err != nil {
		return PasswordConfig{}, nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return PasswordConfig{}, nil, nil, fmt.Errorf("invalid salt: %w", err)
	}
	cfg.SaltLen = uint32(len(salt))

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return PasswordConfig{}, nil, nil, fmt.Errorf("invalid hash: %w", err)
	}
	cfg.KeyLen = uint32(len(hash))

	return cfg, salt, hash, nil
}
