package crypto

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestNewEncryptor_ValidKeySizes(t *testing.T) {
	for _, size := range []int{16, 24, 32} {
		key := make([]byte, size)
		enc, err := NewEncryptor(key)
		if err != nil {
			t.Errorf("NewEncryptor with %d-byte key should succeed, got error: %v", size, err)
		}
		if enc == nil {
			t.Errorf("NewEncryptor with %d-byte key should return non-nil encryptor", size)
		}
	}
}

func TestNewEncryptor_InvalidKeySize(t *testing.T) {
	key := make([]byte, 15)
	_, err := NewEncryptor(key)
	if err == nil {
		t.Error("NewEncryptor with 15-byte key should return error")
	}
	if !errors.Is(err, ErrInvalidKeySize) {
		t.Errorf("Expected ErrInvalidKeySize, got: %v", err)
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	plaintexts := []string{
		"",
		"hello world",
		"my-super-secret-password-123!",
		"Multi\nline\ntext\twith\ttabs",
		"日本語のテキスト",
		"{}[]{}{}{}",
	}

	for _, pt := range plaintexts {
		ct, err := enc.Encrypt(pt)
		if err != nil {
			t.Errorf("Encrypt(%q) returned error: %v", pt, err)
			continue
		}

		if pt != "" && ct == "" {
			t.Errorf("Encrypt(%q) returned empty ciphertext for non-empty plaintext", pt)
		}

		decrypted, err := enc.Decrypt(ct)
		if err != nil {
			t.Errorf("Decrypt returned error: %v", err)
			continue
		}

		if decrypted != pt {
			t.Errorf("Round trip failed: expected %q, got %q", pt, decrypted)
		}
	}
}

func TestEncrypt_ProducesUniqueCiphertext(t *testing.T) {
	key := make([]byte, 32)
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	pt := "secret value"
	ct1, _ := enc.Encrypt(pt)
	ct2, _ := enc.Encrypt(pt)

	if ct1 == ct2 {
		t.Error("Two encryptions of the same plaintext should produce different ciphertext (random nonce)")
	}
}

func TestEncrypt_EmptyString(t *testing.T) {
	key := make([]byte, 32)
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	ct, err := enc.Encrypt("")
	if err != nil {
		t.Errorf("Encrypt(\"\") returned error: %v", err)
	}
	if ct != "" {
		t.Errorf("Encrypt(\"\") should return empty string, got %q", ct)
	}

	pt, err := enc.Decrypt("")
	if err != nil {
		t.Errorf("Decrypt(\"\") returned error: %v", err)
	}
	if pt != "" {
		t.Errorf("Decrypt(\"\") should return empty string, got %q", pt)
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	key := make([]byte, 32)
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	_, err = enc.Decrypt("not-valid-base64!!!")
	if err == nil {
		t.Error("Decrypt with invalid base64 should return error")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	key := make([]byte, 32)
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// Nonce size is 12 for GCM; shorter should fail
	short := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err = enc.Decrypt(short)
	if err == nil {
		t.Error("Decrypt with too-short ciphertext should return error")
	}
	if err != ErrInvalidCiphertext {
		t.Errorf("Expected ErrInvalidCiphertext, got: %v", err)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 0xFF

	enc1, _ := NewEncryptor(key1)
	enc2, _ := NewEncryptor(key2)

	ct, _ := enc1.Encrypt("secret")
	_, err := enc2.Decrypt(ct)
	if err == nil {
		t.Error("Decrypt with wrong key should return error")
	}
}

func TestEncrypt_TamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := NewEncryptor(key)

	ct, _ := enc.Encrypt("original value")

	// Decode, tamper, re-encode
	data, _ := base64.StdEncoding.DecodeString(ct)
	data[len(data)-1] ^= 0xFF
	tampered := base64.StdEncoding.EncodeToString(data)

	_, err := enc.Decrypt(tampered)
	if err == nil {
		t.Error("Decrypt with tampered ciphertext should return error")
	}
}
