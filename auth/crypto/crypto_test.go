package crypto

import (
	"context"
	"strings"
	"testing"

	"github.com/locky/auth/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordHasher_Hash(t *testing.T) {
	hasher := NewPasswordHasher()

	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "simple_password",
			password: "password123",
		},
		{
			name:     "complex_password",
			password: "MyP@ssw0rd!2024",
		},
		{
			name:     "long_password",
			password: strings.Repeat("a", 100),
		},
		{
			name:     "password_with_special_chars",
			password: "!@#$%^&*()_+-=[]{}|;:,.<>?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hasher.Hash(tt.password)
			require.NoError(t, err)
			require.NotEmpty(t, hash)

			// Verify hash format (should be argon2id)
			assert.True(t, strings.HasPrefix(hash, "$argon2id$"))

			// Verify password matches hash
			match, err := hasher.Verify(tt.password, hash)
			require.NoError(t, err)
			assert.True(t, match)

			// Verify wrong password doesn't match
			match, err = hasher.Verify(tt.password+"wrong", hash)
			require.NoError(t, err)
			assert.False(t, match)
		})
	}
}

func TestPasswordHasher_Verify_InvalidHash(t *testing.T) {
	hasher := NewPasswordHasher()

	tests := []struct {
		name    string
		hash    string
		wantErr bool
	}{
		{
			name:    "empty_hash",
			hash:    "",
			wantErr: true,
		},
		{
			name:    "invalid_format",
			hash:    "not-a-valid-hash",
			wantErr: true,
		},
		{
			name:    "wrong_algorithm",
			hash:    "$argon2i$v=19$m=65536,t=3,p=4$c2FsdA$hash",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := hasher.Verify("password", tt.hash)
			if tt.wantErr {
				assert.Error(t, err)
				assert.False(t, match)
			} else {
				require.NoError(t, err)
				assert.False(t, match)
			}
		})
	}
}

func TestPasswordHasher_DifferentHashes(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "same_password"

	// Hash the same password twice - should get different hashes (due to salt)
	hash1, err := hasher.Hash(password)
	require.NoError(t, err)

	hash2, err := hasher.Hash(password)
	require.NoError(t, err)

	// Hashes should be different
	assert.NotEqual(t, hash1, hash2)

	// But both should verify correctly
	match1, err := hasher.Verify(password, hash1)
	require.NoError(t, err)
	assert.True(t, match1)

	match2, err := hasher.Verify(password, hash2)
	require.NoError(t, err)
	assert.True(t, match2)
}

func TestHashString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:  "simple_string",
			input: "test",
		},
		{
			name:  "empty_string",
			input: "",
		},
		{
			name:  "long_string",
			input: strings.Repeat("a", 1000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := HashString(tt.input)
			hash2 := HashString(tt.input)

			// Same input should produce same hash
			assert.Equal(t, hash1, hash2)

			// Hash should not be empty
			assert.NotEmpty(t, hash1)

			// Different input should produce different hash
			if tt.input != "" {
				differentHash := HashString(tt.input + "different")
				assert.NotEqual(t, hash1, differentHash)
			}
		})
	}
}

// Mock implementations for testing JWT Manager

type mockSigningKeyStore struct {
	keys map[string]*core.SigningKey
}

func newMockSigningKeyStore() *mockSigningKeyStore {
	return &mockSigningKeyStore{
		keys: make(map[string]*core.SigningKey),
	}
}

func (m *mockSigningKeyStore) Create(ctx context.Context, key *core.SigningKey) error {
	m.keys[key.ID] = key
	return nil
}

func (m *mockSigningKeyStore) GetActive(ctx context.Context, tenantID string) (*core.SigningKey, error) {
	for _, key := range m.keys {
		if key.TenantID == tenantID && key.Status == "active" {
			return key, nil
		}
	}
	return nil, assert.AnError
}

func (m *mockSigningKeyStore) GetByKID(ctx context.Context, tenantID, kid string) (*core.SigningKey, error) {
	for _, key := range m.keys {
		if key.TenantID == tenantID && key.KID == kid {
			return key, nil
		}
	}
	return nil, assert.AnError
}

func (m *mockSigningKeyStore) ListActive(ctx context.Context, tenantID string) ([]*core.SigningKey, error) {
	var result []*core.SigningKey
	for _, key := range m.keys {
		if key.TenantID == tenantID && (key.Status == "active" || key.Status == "inactive") {
			result = append(result, key)
		}
	}
	return result, nil
}

func (m *mockSigningKeyStore) MarkInactive(ctx context.Context, tenantID, id string) error {
	if key, ok := m.keys[id]; ok {
		key.Status = "inactive"
	}
	return nil
}

func (m *mockSigningKeyStore) MarkRetired(ctx context.Context, tenantID, id string) error {
	if key, ok := m.keys[id]; ok {
		key.Status = "retired"
	}
	return nil
}

func TestKeyManager_GenerateKey(t *testing.T) {
	store := newMockSigningKeyStore()
	manager := NewKeyManager(store, nil)

	tenantID := "tenant-123"
	key, err := manager.GenerateKey(context.Background(), tenantID)

	require.NoError(t, err)
	require.NotNil(t, key)
	assert.NotEmpty(t, key.ID)
	assert.Equal(t, tenantID, key.TenantID)
	assert.NotEmpty(t, key.KID)
	assert.NotEmpty(t, key.PublicJWK)
	assert.NotEmpty(t, key.PrivateKeyEncrypted)
	assert.Equal(t, "active", key.Status)
	assert.False(t, key.CreatedAt.IsZero())
	assert.False(t, key.NotBefore.IsZero())
	assert.False(t, key.NotAfter.IsZero())
	assert.True(t, key.NotAfter.After(key.NotBefore))
}

func TestKeyManager_GetPublicJWKS(t *testing.T) {
	store := newMockSigningKeyStore()
	manager := NewKeyManager(store, nil)

	tenantID := "tenant-123"

	// Generate some keys
	for i := 0; i < 3; i++ {
		_, err := manager.GenerateKey(context.Background(), tenantID)
		require.NoError(t, err)
	}

	jwks, err := manager.GetPublicJWKS(context.Background(), tenantID)
	require.NoError(t, err)
	require.NotNil(t, jwks)

	keys, ok := jwks["keys"].([]map[string]interface{})
	require.True(t, ok)
	assert.Len(t, keys, 3)

	// Verify JWK structure
	for _, jwk := range keys {
		assert.NotEmpty(t, jwk["kty"])
		assert.NotEmpty(t, jwk["kid"])
		assert.Equal(t, "EC", jwk["kty"])
	}
}

func TestKeyManager_GetPublicJWKS_NoKeys(t *testing.T) {
	store := newMockSigningKeyStore()
	manager := NewKeyManager(store, nil)

	tenantID := "tenant-no-keys"

	jwks, err := manager.GetPublicJWKS(context.Background(), tenantID)
	require.NoError(t, err)
	require.NotNil(t, jwks)

	keys, ok := jwks["keys"].([]map[string]interface{})
	require.True(t, ok)
	assert.Empty(t, keys)
}

func TestEncryptDecryptPrivateKey(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
		key       []byte
		wantErr   bool
	}{
		{
			name:      "valid_encryption",
			plaintext: []byte("test private key data"),
			key:       make([]byte, 32),
			wantErr:   false,
		},
		{
			name:      "nil_key_no_encryption",
			plaintext: []byte("test private key data"),
			key:       nil,
			wantErr:   false,
		},
		{
			name:      "empty_plaintext",
			plaintext: []byte{},
			key:       make([]byte, 32),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encryptPrivateKey(tt.plaintext, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			decrypted, err := decryptPrivateKey(encrypted, tt.key)
			require.NoError(t, err)

			// Empty plaintext may decrypt to nil; treat as equivalent to []byte{}
			if len(tt.plaintext) == 0 {
				assert.Empty(t, decrypted)
			} else {
				assert.Equal(t, tt.plaintext, decrypted)
			}
		})
	}
}

func TestEncryptDecryptPrivateKey_InvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)

	// Too short ciphertext
	_, err := decryptPrivateKey([]byte("short"), key)
	assert.Error(t, err)

	// Invalid ciphertext (garbage)
	_, err = decryptPrivateKey([]byte(strings.Repeat("a", 50)), key)
	assert.Error(t, err)
}

func TestKeyManager_Sign(t *testing.T) {
	store := newMockSigningKeyStore()
	manager := NewKeyManager(store, nil)

	tenantID := "tenant-123"
	key, err := manager.GenerateKey(context.Background(), tenantID)
	require.NoError(t, err)
	assert.NotNil(t, key)

	claims := map[string]interface{}{
		"sub":   "user-123",
		"email": "test@example.com",
	}

	token, err := manager.Sign(context.Background(), tenantID, claims)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Token should be in JWT format (3 parts separated by dots)
	parts := strings.Split(token, ".")
	assert.Len(t, parts, 3)
}
