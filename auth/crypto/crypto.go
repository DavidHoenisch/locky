package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/locky/auth/core"
	"golang.org/x/crypto/argon2"
)

const (
	// Argon2id parameters
	argon2Time    = 3
	argon2Memory  = 64 * 1024
	argon2Threads = 4
	argon2KeyLen  = 32

	// Key encryption
	keyEncryptionKeyLen   = 32
	keyEncryptionNonceLen = 12
)

// PasswordHasher handles password hashing and verification
type PasswordHasher struct{}

// NewPasswordHasher creates a new PasswordHasher
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{}
}

// Hash generates an Argon2id hash of the password
func (h *PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Encode as: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory, argon2Time, argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash))

	return encoded, nil
}

// Verify checks if a password matches the given hash
func (h *PasswordHasher) Verify(password, encodedHash string) (bool, error) {
	// Parse the encoded hash: $argon2id$v=19$m=65536,t=3,p=4$<salt_b64>$<hash_b64>
	// Sscanf %s stops at whitespace, so split by $ to get salt and hash reliably
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false, fmt.Errorf("parse hash: invalid format")
	}
	var memory, argon2TimeParam uint32
	var threads uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &argon2TimeParam, &threads)
	if err != nil {
		return false, fmt.Errorf("parse hash: %w", err)
	}
	saltB64 := parts[4]
	hashB64 := parts[5]

	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return false, fmt.Errorf("decode salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argon2TimeParam, memory, threads, argon2KeyLen)

	expectedHash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}

	// Constant-time comparison
	if len(hash) != len(expectedHash) {
		return false, nil
	}

	var result byte
	for i := range hash {
		result |= hash[i] ^ expectedHash[i]
	}

	return result == 0, nil
}

// JWTManager handles JWT operations
type JWTManager struct {
	keys core.SigningKeyStore
}

// NewJWTManager creates a new JWTManager
func NewJWTManager(keys core.SigningKeyStore) *JWTManager {
	return &JWTManager{keys: keys}
}

// Sign creates a JWT for the given tenant with the specified claims
func (m *JWTManager) Sign(ctx context.Context, tenantID, issuer string, claims map[string]interface{}, ttl time.Duration) (string, error) {
	key, err := m.keys.GetActive(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("get active key: %w", err)
	}

	// Decrypt private key
	privateKey, err := decryptPrivateKey(key.PrivateKeyEncrypted, nil) // TODO: provide master key
	if err != nil {
		return "", fmt.Errorf("decrypt private key: %w", err)
	}

	now := time.Now()
	tokenClaims := jwt.MapClaims{
		"iss": issuer,
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(ttl).Unix(),
		"jti": uuid.New().String(),
		"tid": tenantID,
	}

	for k, v := range claims {
		tokenClaims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, tokenClaims)
	token.Header["kid"] = key.KID

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

// Verify validates a JWT and returns its claims
func (m *JWTManager) Verify(ctx context.Context, tenantID, tokenString string) (*core.TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid header")
		}

		key, err := m.keys.GetByKID(ctx, tenantID, kid)
		if err != nil {
			return nil, fmt.Errorf("get key: %w", err)
		}

		var jwk map[string]interface{}
		if err := json.Unmarshal(key.PublicJWK, &jwk); err != nil {
			return nil, fmt.Errorf("parse jwk: %w", err)
		}

		// Extract public key from JWK (simplified - assumes EC key)
		xB64, _ := jwk["x"].(string)
		yB64, _ := jwk["y"].(string)
		crv, _ := jwk["crv"].(string)

		xBytes, _ := base64.RawURLEncoding.DecodeString(xB64)
		yBytes, _ := base64.RawURLEncoding.DecodeString(yB64)

		var curve elliptic.Curve
		switch crv {
		case "P-256":
			curve = elliptic.P256()
		case "P-384":
			curve = elliptic.P384()
		case "P-521":
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("unsupported curve: %s", crv)
		}

		x, y := new(big.Int).SetBytes(xBytes), new(big.Int).SetBytes(yBytes)
		return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
	}, jwt.WithIssuer(tenantID))

	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	tc := &core.TokenClaims{}
	if sub, ok := claims["sub"].(string); ok {
		tc.Subject = sub
	}
	if iss, ok := claims["iss"].(string); ok {
		tc.Issuer = iss
	}
	if aud, ok := claims["aud"].(string); ok {
		tc.Audience = aud
	}
	if tid, ok := claims["tid"].(string); ok {
		tc.TenantID = tid
	}
	if sid, ok := claims["sid"].(string); ok {
		tc.SessionID = &sid
	}
	if scope, ok := claims["scope"].(string); ok {
		tc.Scope = scope
	}
	if jti, ok := claims["jti"].(string); ok {
		tc.JWTID = jti
	}
	if iat, ok := claims["iat"].(float64); ok {
		tc.IssuedAt = int64(iat)
	}
	if exp, ok := claims["exp"].(float64); ok {
		tc.ExpiresAt = int64(exp)
	}
	if nbf, ok := claims["nbf"].(float64); ok {
		tc.NotBefore = int64(nbf)
	}
	if roles, ok := claims["roles"].([]interface{}); ok {
		tc.Roles = make([]string, len(roles))
		for i, r := range roles {
			tc.Roles[i] = r.(string)
		}
	}

	return tc, nil
}

// KeyManager handles signing key operations
type KeyManager struct {
	keys      core.SigningKeyStore
	masterKey []byte
}

// NewKeyManager creates a new KeyManager
func NewKeyManager(keys core.SigningKeyStore, masterKey []byte) *KeyManager {
	return &KeyManager{
		keys:      keys,
		masterKey: masterKey,
	}
}

// GenerateKey generates a new signing key for a tenant
func (m *KeyManager) GenerateKey(ctx context.Context, tenantID string) (*core.SigningKey, error) {
	// Generate EC key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	kid := uuid.New().String()

	// Create JWK
	publicKey := &privateKey.PublicKey
	xBytes := publicKey.X.Bytes()
	yBytes := publicKey.Y.Bytes()

	jwk := map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"kid": kid,
		"x":   base64.RawURLEncoding.EncodeToString(xBytes),
		"y":   base64.RawURLEncoding.EncodeToString(yBytes),
		"use": "sig",
	}

	jwkJSON, err := json.Marshal(jwk)
	if err != nil {
		return nil, fmt.Errorf("marshal jwk: %w", err)
	}

	// Serialize and encrypt private key
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}

	encryptedKey, err := encryptPrivateKey(privateKeyBytes, m.masterKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt private key: %w", err)
	}

	key := &core.SigningKey{
		ID:                  uuid.New().String(),
		TenantID:            tenantID,
		KID:                 kid,
		PublicJWK:           jwkJSON,
		PrivateKeyEncrypted: encryptedKey,
		Status:              "active",
		CreatedAt:           time.Now(),
		NotBefore:           time.Now(),
		NotAfter:            time.Now().Add(90 * 24 * time.Hour), // 90 days default
	}

	if err := m.keys.Create(ctx, key); err != nil {
		return nil, fmt.Errorf("store key: %w", err)
	}

	return key, nil
}

// GetPublicJWKS returns the JWKS for a tenant
func (m *KeyManager) GetPublicJWKS(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	keys, err := m.keys.ListActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list keys: %w", err)
	}

	jwks := make([]map[string]interface{}, 0, len(keys))
	for _, key := range keys {
		var jwk map[string]interface{}
		if err := json.Unmarshal(key.PublicJWK, &jwk); err != nil {
			continue
		}
		jwks = append(jwks, jwk)
	}

	return map[string]interface{}{"keys": jwks}, nil
}

// Sign signs claims and returns a JWT string
func (m *KeyManager) Sign(ctx context.Context, tenantID string, claims map[string]interface{}) (string, error) {
	key, err := m.keys.GetActive(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("get active key: %w", err)
	}

	// Decrypt private key
	privateKeyBytes, err := decryptPrivateKey(key.PrivateKeyEncrypted, m.masterKey)
	if err != nil {
		return "", fmt.Errorf("decrypt private key: %w", err)
	}

	privateKey, err := x509.ParseECPrivateKey(privateKeyBytes)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}

	// Create token
	tokenClaims := jwt.MapClaims{}
	for k, v := range claims {
		tokenClaims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, tokenClaims)
	token.Header["kid"] = key.KID

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

// encryptPrivateKey encrypts a private key using AES-GCM
func encryptPrivateKey(plaintext, key []byte) ([]byte, error) {
	if key == nil {
		// For now, return plaintext if no master key provided
		return plaintext, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, keyEncryptionNonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptPrivateKey decrypts a private key using AES-GCM
func decryptPrivateKey(ciphertext, key []byte) ([]byte, error) {
	if key == nil {
		// For now, return ciphertext if no master key provided
		return ciphertext, nil
	}

	if len(ciphertext) < keyEncryptionNonceLen {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:keyEncryptionNonceLen]
	ciphertext = ciphertext[keyEncryptionNonceLen:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// HashString creates a SHA256 hash of a string (for token hashing)
func HashString(s string) string {
	hash := sha256.Sum256([]byte(s))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
