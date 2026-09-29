package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

type TOTPEncryptionService struct {
	aead cipher.AEAD
}

func NewTOTPEncryptionService(secret string) (*TOTPEncryptionService, error) {
	if len([]byte(secret)) < 32 {
		return nil, errors.New(
			"TOTP encryption key must be at least 32 bytes",
		)
	}

	key := sha256.Sum256([]byte(secret))

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create TOTP encryption cipher: %w",
			err,
		)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create TOTP encryption AEAD: %w",
			err,
		)
	}

	return &TOTPEncryptionService{
		aead: aead,
	}, nil
}

func (s *TOTPEncryptionService) Encrypt(
	plaintext string,
) (string, error) {
	if plaintext == "" {
		return "", errors.New("plaintext cannot be empty")
	}

	nonce := make([]byte, s.aead.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf(
			"failed to generate encryption nonce: %w",
			err,
		)
	}

	ciphertext := s.aead.Seal(
		nonce,
		nonce,
		[]byte(plaintext),
		nil,
	)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *TOTPEncryptionService) Decrypt(
	encodedCiphertext string,
) (string, error) {
	if encodedCiphertext == "" {
		return "", errors.New("ciphertext cannot be empty")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(
		encodedCiphertext,
	)
	if err != nil {
		return "", fmt.Errorf(
			"failed to decode encrypted TOTP secret: %w",
			err,
		)
	}

	nonceSize := s.aead.NonceSize()

	if len(ciphertext) < nonceSize {
		return "", errors.New(
			"invalid encrypted TOTP secret",
		)
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	plaintext, err := s.aead.Open(
		nil,
		nonce,
		ciphertext,
		nil,
	)
	if err != nil {
		return "", errors.New(
			"failed to decrypt TOTP secret",
		)
	}

	return string(plaintext), nil
}
