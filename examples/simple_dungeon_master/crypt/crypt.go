package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// payload represents encrypted data with turn number for replay protection
type payload struct {
	Data json.RawMessage `json:"data"` // Encrypted data
	Turn int             `json:"turn"` // Turn number for replay protection
}

// State manages cryptographic operations for skill check validation
type State struct {
	secret []byte // Server secret for AES-256 encryption (32 bytes)
}

// NewState creates a new crypt State with a randomly generated secret
func NewState() (*State, error) {
	secret, err := generateSecret()
	if err != nil {
		return nil, err
	}
	return &State{secret: secret}, nil
}

// generateSecret creates a 32-byte random secret for AES-256 encryption
func generateSecret() ([]byte, error) {
	secret := make([]byte, 32) // AES-256 requires 32-byte key
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}
	return secret, nil
}

// Encrypt encrypts any JSON-serializable data with turn number for tamper protection
func (s *State) Encrypt(data any, turn int) (string, error) {
	return encrypt(s.secret, data, turn)
}

// Decrypt decrypts and validates encrypted data, unmarshaling into the provided result
func (s *State) Decrypt(encrypted string, expectedTurn int, result any) error {
	return decrypt(s.secret, encrypted, expectedTurn, result)
}

// encrypt encrypts any data using AES-GCM authenticated encryption
func encrypt(secret []byte, data any, turn int) (string, error) {
	// Marshal data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// Create payload
	p := payload{
		Data: dataJSON,
		Turn: turn,
	}

	// Marshal payload to JSON
	plaintext, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(secret)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode (provides authenticated encryption)
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Return base64 encoded
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts and validates encrypted data
func decrypt(secret []byte, encrypted string, expectedTurn int, result any) error {
	// Decode base64
	cipherText, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return fmt.Errorf("invalid encryption encoding: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(secret)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return errors.New("ciphertext too short")
	}
	nonce, cipherText := cipherText[:nonceSize], cipherText[nonceSize:]

	// Decrypt and verify authentication tag
	plaintext, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return errors.New("decryption failed: invalid or tampered data")
	}

	// Unmarshal payload
	var p payload
	if err := json.Unmarshal(plaintext, &p); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Validate turn number (replay protection)
	if p.Turn != expectedTurn {
		return fmt.Errorf("invalid turn number: encrypted data is for turn %d but expected turn %d", p.Turn, expectedTurn)
	}

	// Unmarshal data into result
	if err := json.Unmarshal(p.Data, result); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}
