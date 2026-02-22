package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

const prefix = "enc:"

var ErrMissingEncryptionKey = errors.New("encryption key is required")

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded
// ciphertext prefixed with "enc:" for identification.
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
// If the value doesn't have the "enc:" prefix, it's returned as-is
// (supports reading legacy unencrypted data).
func Decrypt(ciphertext, key string) (string, error) {
	if ciphertext == "" {
		return ciphertext, nil
	}

	// Not encrypted — return as-is (legacy data)
	if len(ciphertext) < len(prefix) || ciphertext[:len(prefix)] != prefix {
		return ciphertext, nil
	}
	if strings.TrimSpace(key) == "" {
		return "", ErrMissingEncryptionKey
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext[len(prefix):])
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(deriveKey(key))
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
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}

// deriveKey pads or truncates the key to exactly 32 bytes for AES-256.
func deriveKey(key string) []byte {
	k := make([]byte, 32)
	copy(k, []byte(key))
	return k
}
