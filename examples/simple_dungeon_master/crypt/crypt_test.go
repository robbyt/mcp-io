package crypt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewState verifies State creation with secret generation
func TestNewState(t *testing.T) {
	t.Parallel()

	state, err := NewState()
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Len(t, state.secret, 32, "Secret should be 32 bytes for AES-256")

	// Verify randomness by generating multiple states
	state2, err := NewState()
	require.NoError(t, err)
	assert.NotEqual(t, state.secret, state2.secret, "Secrets should be random and unique")
}

// TestEncryptDecrypt_BasicRoundTrip verifies basic encryption and decryption
func TestEncryptDecrypt_BasicRoundTrip(t *testing.T) {
	t.Parallel()

	state, err := NewState()
	require.NoError(t, err)

	// Test with roll data
	originalData := map[string]int{"result": 15}
	turnNumber := 5

	// Encrypt
	encrypted, err := state.Encrypt(originalData, turnNumber)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	// Decrypt
	var decrypted map[string]int
	err = state.Decrypt(encrypted, turnNumber, &decrypted)
	require.NoError(t, err)
	assert.Equal(t, originalData, decrypted)
}

// TestEncrypt_DifferentTypes verifies encryption works with different data types
func TestEncrypt_DifferentTypes(t *testing.T) {
	t.Parallel()

	state, err := NewState()
	require.NoError(t, err)

	tests := []struct {
		name string
		data any
	}{
		{"map", map[string]int{"result": 18}},
		{"struct", struct {
			Result int
			Name   string
		}{Result: 10, Name: "test"}},
		{"slice", []int{1, 2, 3}},
		{"string", "test string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := state.Encrypt(tt.data, 1)
			require.NoError(t, err)
			assert.NotEmpty(t, encrypted)
		})
	}
}

// TestDecrypt_WrongTurnNumber verifies turn number validation
func TestDecrypt_WrongTurnNumber(t *testing.T) {
	t.Parallel()

	state, err := NewState()
	require.NoError(t, err)

	data := map[string]int{"result": 12}

	// Encrypt with turn 5
	encrypted, err := state.Encrypt(data, 5)
	require.NoError(t, err)

	// Try to decrypt with wrong turn number
	var decrypted map[string]int
	err = state.Decrypt(encrypted, 10, &decrypted)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid turn number")
}

// TestDecrypt_TamperedData verifies tamper detection
func TestDecrypt_TamperedData(t *testing.T) {
	t.Parallel()

	state, err := NewState()
	require.NoError(t, err)

	data := map[string]int{"result": 5}

	// Encrypt
	encrypted, err := state.Encrypt(data, 1)
	require.NoError(t, err)

	// Tamper with the encrypted data by flipping bits in the middle
	tamperedEncrypted := encrypted[:len(encrypted)/2] + "TAMPERED" + encrypted[len(encrypted)/2+8:]

	// Decryption should fail
	var decrypted map[string]int
	err = state.Decrypt(tamperedEncrypted, 1, &decrypted)
	require.Error(t, err)
}

// TestDecrypt_WrongSecret verifies different secret fails
func TestDecrypt_WrongSecret(t *testing.T) {
	t.Parallel()

	state1, err := NewState()
	require.NoError(t, err)

	state2, err := NewState()
	require.NoError(t, err)

	data := map[string]int{"result": 15}

	// Encrypt with state1
	encrypted, err := state1.Encrypt(data, 1)
	require.NoError(t, err)

	// Try to decrypt with state2 (different secret)
	var decrypted map[string]int
	err = state2.Decrypt(encrypted, 1, &decrypted)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decryption failed")
}

// TestDecrypt_InvalidFormat verifies malformed data fails
func TestDecrypt_InvalidFormat(t *testing.T) {
	t.Parallel()

	state, err := NewState()
	require.NoError(t, err)

	tests := []struct {
		name      string
		encrypted string
	}{
		{"empty string", ""},
		{"invalid base64", "!!!invalid!!!"},
		{"too short", "YWJj"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var decrypted map[string]int
			err := state.Decrypt(tt.encrypted, 1, &decrypted)
			assert.Error(t, err)
		})
	}
}

// TestEncrypt_UniqueOutputs verifies each encryption produces unique ciphertext
func TestEncrypt_UniqueOutputs(t *testing.T) {
	t.Parallel()

	state, err := NewState()
	require.NoError(t, err)

	data := map[string]int{"result": 10}

	// Encrypt same data twice
	encrypted1, err := state.Encrypt(data, 1)
	require.NoError(t, err)

	encrypted2, err := state.Encrypt(data, 1)
	require.NoError(t, err)

	// Outputs should be different due to random nonce
	assert.NotEqual(t, encrypted1, encrypted2, "Each encryption should produce unique ciphertext")

	// But both should decrypt to same value
	var decrypted1, decrypted2 map[string]int
	require.NoError(t, state.Decrypt(encrypted1, 1, &decrypted1))
	require.NoError(t, state.Decrypt(encrypted2, 1, &decrypted2))
	assert.Equal(t, decrypted1, decrypted2)
}

// TestEncryptDecrypt_AllDiceRolls verifies all valid dice rolls (1-20)
func TestEncryptDecrypt_AllDiceRolls(t *testing.T) {
	t.Parallel()

	state, err := NewState()
	require.NoError(t, err)

	// Test all valid dice results
	for result := 1; result <= 20; result++ {
		data := map[string]int{"result": result}

		encrypted, err := state.Encrypt(data, 1)
		require.NoError(t, err)

		var decrypted map[string]int
		err = state.Decrypt(encrypted, 1, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, result, decrypted["result"], "Result should match for roll %d", result)
	}
}

// TestEncryptDecrypt_MultipleTurns verifies turn number is part of encryption
func TestEncryptDecrypt_MultipleTurns(t *testing.T) {
	t.Parallel()

	state, err := NewState()
	require.NoError(t, err)

	// Encrypt same data for different turns
	data := map[string]int{"result": 15}

	for turn := 0; turn < 10; turn++ {
		encrypted, err := state.Encrypt(data, turn)
		require.NoError(t, err)

		// Should decrypt successfully with correct turn
		var decrypted map[string]int
		err = state.Decrypt(encrypted, turn, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, data, decrypted)

		// Should fail with wrong turn
		err = state.Decrypt(encrypted, turn+1, &decrypted)
		require.Error(t, err)
	}
}
