package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"crypto/rand"
	"encoding/hex"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

const (
	legacyPrefix = "enc:"
	prefix       = "enc2:"
)

var ErrMissingEncryptionKey = errors.New("encryption key is required")

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded
// ciphertext prefixed with "enc2:" for identification.
func Encrypt(plaintext, key string) (string, error) {
	if plaintext == "" {
		return plaintext, nil
	}
	if strings.TrimSpace(key) == "" {
		return "", ErrMissingEncryptionKey
	}

	block, err := aes.NewCipher(deriveKey(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a value previously encrypted with Encrypt.
// If the value doesn't have a recognized encryption prefix, it's returned as-is
// (supports reading legacy unencrypted data).
func Decrypt(ciphertext, key string) (string, error) {
	if ciphertext == "" {
		return ciphertext, nil
	}

	// Not encrypted — return as-is (legacy data)
	if !strings.HasPrefix(ciphertext, prefix) && !strings.HasPrefix(ciphertext, legacyPrefix) {
		return ciphertext, nil
	}
	if strings.TrimSpace(key) == "" {
		return "", ErrMissingEncryptionKey
	}

	keyBytes := deriveKey(key)
	payload := ciphertext
	if strings.HasPrefix(ciphertext, legacyPrefix) {
		keyBytes = deriveLegacyKey(key)
		payload = ciphertext[len(legacyPrefix):]
	} else {
		payload = ciphertext[len(prefix):]
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// DecryptFields decrypts multiple string fields in place using the given key.
// Each field pointer is updated with its decrypted value if decryption succeeds;
// otherwise the original value is preserved (supports legacy unencrypted data).
func DecryptFields(key string, fields ...*string) {
	for _, f := range fields {
		if dec, err := Decrypt(*f, key); err == nil {
			*f = dec
		}
	}
}

// IsEncrypted checks if a value has the encryption prefix.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, prefix) || strings.HasPrefix(value, legacyPrefix)
}

// deriveKey normalizes operator-provided secrets into a stable 32-byte AES key.
// It accepts raw 32-byte values encoded as hex/base64 and falls back to hashing
// arbitrary passphrases for forward compatibility.
func deriveKey(key string) []byte {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return make([]byte, 32)
	}

	if decoded, err := hex.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded
	}
	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded
	}

	sum := sha256.Sum256([]byte(trimmed))
	return sum[:]
}

// deriveLegacyKey preserves decryption for existing enc: ciphertexts that used
// naive truncation/padding of the configured secret.
func deriveLegacyKey(key string) []byte {
	k := make([]byte, 32)
	copy(k, []byte(key))
	return k
}
