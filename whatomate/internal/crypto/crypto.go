package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	legacyPrefix = "enc:"
	prefixV2     = "enc2:"
	prefixV3     = "enc3:"

	argon2Time    = 3
	argon2Memory  = 64 * 1024
	argon2Threads = 2
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

var ErrMissingEncryptionKey = errors.New("encryption key is required")
var ErrLegacyEncryptionDisabled = errors.New("legacy encryption format is disabled")
var ErrNotEncrypted = errors.New("value is not encrypted")

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded
// ciphertext prefixed with "enc3:" for identification.
func Encrypt(plaintext, key string) (string, error) {
	if plaintext == "" {
		return plaintext, nil
	}

	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", ErrMissingEncryptionKey
	}

	rawKey, ok := decodeRawKey(trimmed)
	saltLen := 0
	var salt []byte
	keyBytes := rawKey
	if !ok {
		saltLen = argon2SaltLen
		salt = make([]byte, saltLen)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return "", err
		}
		keyBytes = deriveKeyV3(trimmed, salt)
	}

	block, err := aes.NewCipher(keyBytes)
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
	payload := make([]byte, 0, 1+saltLen+len(ciphertext))
	payload = append(payload, byte(saltLen))
	if saltLen > 0 {
		payload = append(payload, salt...)
	}
	payload = append(payload, ciphertext...)

	return prefixV3 + base64.StdEncoding.EncodeToString(payload), nil
}

// Decrypt decrypts a value previously encrypted with Encrypt.
// If the value doesn't have a recognized encryption prefix, it's returned as-is
// (supports reading legacy unencrypted data).
func Decrypt(ciphertext, key string) (string, error) {
	return DecryptWithPolicy(ciphertext, key, true)
}

// DecryptStrict decrypts a value that must be encrypted (enc3:/enc2:/enc: prefix required).
func DecryptStrict(ciphertext, key string) (string, error) {
	if ciphertext == "" {
		return ciphertext, nil
	}
	if !IsEncrypted(ciphertext) {
		return "", ErrNotEncrypted
	}
	return Decrypt(ciphertext, key)
}

// DecryptWithPolicy decrypts a value previously encrypted with Encrypt.
// If allowLegacy is false, legacy enc:/enc2: payloads are rejected.
func DecryptWithPolicy(ciphertext, key string, allowLegacy bool) (string, error) {
	if ciphertext == "" {
		return ciphertext, nil
	}

	// Not encrypted — return as-is (legacy data)
	if !strings.HasPrefix(ciphertext, prefixV3) && !strings.HasPrefix(ciphertext, prefixV2) && !strings.HasPrefix(ciphertext, legacyPrefix) {
		return ciphertext, nil
	}

	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", ErrMissingEncryptionKey
	}

	if strings.HasPrefix(ciphertext, prefixV3) {
		payload := ciphertext[len(prefixV3):]
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return "", err
		}
		if len(data) < 1 {
			return "", errors.New("ciphertext too short")
		}
		saltLen := int(data[0])
		if len(data) < 1+saltLen {
			return "", errors.New("ciphertext too short")
		}
		salt := data[1 : 1+saltLen]
		ciphertextBytes := data[1+saltLen:]

		var keyBytes []byte
		if saltLen > 0 {
			keyBytes = deriveKeyV3(trimmed, salt)
		} else {
			rawKey, ok := decodeRawKey(trimmed)
			if !ok {
				return "", errors.New("raw encryption key required for enc3 payload")
			}
			keyBytes = rawKey
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
		if len(ciphertextBytes) < nonceSize {
			return "", errors.New("ciphertext too short")
		}

		nonce, ciphertextBytes := ciphertextBytes[:nonceSize], ciphertextBytes[nonceSize:]
		plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
		if err != nil {
			return "", err
		}

		return string(plaintext), nil
	}

	if !allowLegacy {
		return "", ErrLegacyEncryptionDisabled
	}

	keyBytes := deriveKeyV2(trimmed)
	payload := ciphertext //nolint:ineffassign // payload is always reassigned below
	if strings.HasPrefix(ciphertext, legacyPrefix) {
		keyBytes = deriveLegacyKey(trimmed)
		payload = ciphertext[len(legacyPrefix):]
	} else {
		payload = ciphertext[len(prefixV2):]
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
	DecryptFieldsWithPolicy(key, true, fields...)
}

// DecryptFieldsWithPolicy decrypts multiple string fields in place using the given key.
// Each field pointer is updated with its decrypted value if decryption succeeds;
// otherwise the original value is preserved.
func DecryptFieldsWithPolicy(key string, allowLegacy bool, fields ...*string) {
	for _, f := range fields {
		if f == nil {
			continue
		}
		if dec, err := DecryptWithPolicy(*f, key, allowLegacy); err == nil {
			*f = dec
		}
	}
}

// DecryptFieldsStrict decrypts multiple string fields in place using the given key.
// Returns the first error encountered.
func DecryptFieldsStrict(key string, fields ...*string) error {
	for _, f := range fields {
		if f == nil {
			continue
		}
		dec, err := DecryptStrict(*f, key)
		if err != nil {
			return err
		}
		*f = dec
	}
	return nil
}

// IsEncrypted checks if a value has the encryption prefix.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, prefixV3) || strings.HasPrefix(value, prefixV2) || strings.HasPrefix(value, legacyPrefix)
}

// deriveKeyV2 normalizes operator-provided secrets into a stable 32-byte AES key.
// It accepts raw 32-byte values encoded as hex/base64 and falls back to hashing
// arbitrary passphrases for legacy enc2 compatibility.
func deriveKeyV2(key string) []byte {
	if decoded, ok := decodeRawKey(key); ok {
		return decoded
	}

	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return make([]byte, 32)
	}

	sum := sha256.Sum256([]byte(trimmed))
	return sum[:]
}

func deriveKeyV3(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
}

func decodeRawKey(key string) ([]byte, bool) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return nil, false
	}
	if decoded, err := hex.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, true
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, true
	}
	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, true
	}
	return nil, false
}

// deriveLegacyKey preserves decryption for existing enc: ciphertexts that used
// naive truncation/padding of the configured secret.
func deriveLegacyKey(key string) []byte {
	k := make([]byte, 32)
	copy(k, []byte(key))
	return k
}
