package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Encryptor provides AES-256-GCM field-level encryption.
type Encryptor struct {
	aead cipher.AEAD
}

var (
	// ErrInvalidKeySize is returned when the key is not 16, 24, or 32 bytes.
	ErrInvalidKeySize = errors.New("encryption key must be 16, 24, or 32 bytes")
	// ErrInvalidCiphertext is returned when decryption fails.
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

// NewEncryptor creates an AES-GCM encryptor from a raw key (16/24/32 bytes).
func NewEncryptor(key []byte) (*Encryptor, error) {
	switch len(key) {
	case 16, 24, 32:
		// valid
	default:
		return nil, fmt.Errorf("%w, got %d bytes", ErrInvalidKeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &Encryptor{aead: aead}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64 string.
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Prepend nonce to ciphertext
	ciphertext := e.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded AES-256-GCM ciphertext.
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	nonceSize := e.aead.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, sealed := data[:nonceSize], data[nonceSize:]
	plaintext, err := e.aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
