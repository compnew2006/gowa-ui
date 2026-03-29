package crypto

import (
	"errors"
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := "my-secret-key-for-testing-12345"
	plaintext := "EAABsbCS1iHgBO..."

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if encrypted == plaintext {
		t.Fatal("Encrypted value should differ from plaintext")
	}
	if !strings.HasPrefix(encrypted, prefixV3) {
		t.Fatalf("encrypted value should use %q prefix, got %q", prefixV3, encrypted)
	}

	if !IsEncrypted(encrypted) {
		t.Fatal("Encrypted value should have an encryption prefix")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("Decrypted value %q != plaintext %q", decrypted, plaintext)
	}
}

func TestDecrypt_LegacyUnencrypted(t *testing.T) {
	key := "my-secret-key"
	legacy := "plain-text-token-without-prefix"

	decrypted, err := Decrypt(legacy, key)
	if err != nil {
		t.Fatalf("Decrypt legacy failed: %v", err)
	}
	if decrypted != legacy {
		t.Fatalf("Legacy value should be returned as-is, got %q", decrypted)
	}
}

func TestEncrypt_EmptyKeyFails(t *testing.T) {
	_, err := Encrypt("some-secret", "")
	if !errors.Is(err, ErrMissingEncryptionKey) {
		t.Fatalf("expected ErrMissingEncryptionKey, got %v", err)
	}
}

func TestDecrypt_EncryptedValueWithoutKeyFails(t *testing.T) {
	encrypted, err := Encrypt("some-secret", "test-key")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	_, err = Decrypt(encrypted, "")
	if !errors.Is(err, ErrMissingEncryptionKey) {
		t.Fatalf("expected ErrMissingEncryptionKey, got %v", err)
	}
}

func TestDecrypt_LegacyValueWithoutKeyStillWorks(t *testing.T) {
	legacy := "plain-text-token-without-prefix"
	decrypted, err := Decrypt(legacy, "")
	if err != nil {
		t.Fatalf("Decrypt legacy failed: %v", err)
	}
	if decrypted != legacy {
		t.Fatalf("Legacy value should be returned as-is, got %q", decrypted)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := "correct-key"
	key2 := "wrong-key"

	encrypted, _ := Encrypt("secret", key1)
	_, err := Decrypt(encrypted, key2)
	if err == nil {
		t.Fatal("Decrypt with wrong key should fail")
	}
}
