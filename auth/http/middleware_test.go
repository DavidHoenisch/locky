package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/locky/auth/core"
	"github.com/stretchr/testify/assert"
)

// Mock TenantResolver
type mockTenantResolver struct {
	tenant *core.Tenant
	err    error
}

func (m *mockTenantResolver) ResolveTenant(ctx context.Context, host string) (*core.Tenant, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tenant, nil
}

// Mock AdminKeyStore
type mockAdminKeyStore struct {
	key *core.AdminKey
	err error
}

func (m *mockAdminKeyStore) Create(ctx context.Context, key *core.AdminKey) error {
	return nil
}

func (m *mockAdminKeyStore) GetByHash(ctx context.Context, hash string) (*core.AdminKey, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.key, nil
}

func (m *mockAdminKeyStore) List(ctx context.Context) ([]*core.AdminKey, error) {
	return nil, nil
}

func (m *mockAdminKeyStore) Delete(ctx context.Context, id string) error {
	return nil
}

// Mock SessionService
type mockSessionService struct {
	session *core.Session
	err     error
}

func (m *mockSessionService) Create(ctx context.Context, tenantID, userID, clientID string, ip, userAgent string) (*core.Session, error) {
	return nil, nil
}

func (m *mockSessionService) Validate(ctx context.Context, tenantID, sessionID string) (*core.Session, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.session, nil
}

func (m *mockSessionService) Revoke(ctx context.Context, tenantID, sessionID string) error {
	return nil
}

func TestTenantMiddleware_Handler(t *testing.T) {
	tests := []struct {
		name           string
		host           string
		setupResolver  func() *mockTenantResolver
		expectedStatus int
		expectedTenant bool
	}{
		{
			name: "valid_tenant",
			host: "acme.auth.example.com",
			setupResolver: func() *mockTenantResolver {
				return &mockTenantResolver{
					tenant: &core.Tenant{
						ID:   "tenant-123",
						Slug: "acme",
						Name: "Acme Corp",
					},
				}
			},
			expectedStatus: http.StatusOK,
			expectedTenant: true,
		},
		{
			name: "tenant_not_found",
			host: "unknown.auth.example.com",
			setupResolver: func() *mockTenantResolver {
				return &mockTenantResolver{
					err: errors.New("tenant not found"),
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedTenant: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := tt.setupResolver()
			middleware := NewTenantMiddleware(resolver)

			handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if tenant is in context
				tenant, ok := GetTenant(r.Context())
				if tt.expectedTenant {
					assert.True(t, ok)
					assert.NotNil(t, tenant)
				} else {
					assert.False(t, ok)
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestAdminAuthMiddleware_Handler(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		setupStore     func() *mockAdminKeyStore
		expectedStatus int
	}{
		{
			name:   "valid_api_key",
			apiKey: "valid-key-123",
			setupStore: func() *mockAdminKeyStore {
				return &mockAdminKeyStore{
					key: &core.AdminKey{
						ID:      "key-123",
						KeyHash: "valid-key-123",
						Name:    "Test Key",
					},
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "missing_api_key",
			apiKey: "",
			setupStore: func() *mockAdminKeyStore {
				return &mockAdminKeyStore{}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "invalid_api_key",
			apiKey: "invalid-key",
			setupStore: func() *mockAdminKeyStore {
				return &mockAdminKeyStore{
					err: errors.New("invalid key"),
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			middleware := NewAdminAuthMiddleware(store, "")

			handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/admin/test", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-Admin-Key", tt.apiKey)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestCORSMiddleware_Handler(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		allowedOrigins []string
		expectCORS     bool
		expectOrigin   string
	}{
		{
			name:           "allowed_origin",
			origin:         "https://app.example.com",
			allowedOrigins: []string{"https://app.example.com", "https://admin.example.com"},
			expectCORS:     true,
			expectOrigin:   "https://app.example.com",
		},
		{
			name:           "wildcard_origin",
			origin:         "https://any.example.com",
			allowedOrigins: []string{"*"},
			expectCORS:     true,
			expectOrigin:   "https://any.example.com",
		},
		{
			name:           "disallowed_origin",
			origin:         "https://evil.com",
			allowedOrigins: []string{"https://app.example.com"},
			expectCORS:     false,
			expectOrigin:   "",
		},
		{
			name:           "empty_origin",
			origin:         "",
			allowedOrigins: []string{"*"},
			expectCORS:     true,
			expectOrigin:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewCORSMiddleware(tt.allowedOrigins)

			handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if tt.expectCORS {
				assert.Equal(t, tt.expectOrigin, rr.Header().Get("Access-Control-Allow-Origin"))
				assert.NotEmpty(t, rr.Header().Get("Access-Control-Allow-Methods"))
				assert.NotEmpty(t, rr.Header().Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
			} else {
				assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestCORSMiddleware_Handler_Preflight(t *testing.T) {
	middleware := NewCORSMiddleware([]string{"*"})

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should not be called for OPTIONS requests
		t.Error("Handler should not be called for OPTIONS requests")
	}))

	req := httptest.NewRequest("OPTIONS", "/oauth2/token", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "https://app.example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, rr.Header().Get("Access-Control-Allow-Methods"))
}

func TestSessionMiddleware_Handler(t *testing.T) {
	tests := []struct {
		name            string
		cookieValue     string
		setupSession    func() *mockSessionService
		expectedSession bool
	}{
		{
			name:        "valid_session",
			cookieValue: "valid-session-id",
			setupSession: func() *mockSessionService {
				return &mockSessionService{
					session: &core.Session{
						ID:       "valid-session-id",
						TenantID: "tenant-123",
						UserID:   "user-456",
					},
				}
			},
			expectedSession: true,
		},
		{
			name:        "invalid_session",
			cookieValue: "invalid-session-id",
			setupSession: func() *mockSessionService {
				return &mockSessionService{
					err: errors.New("session not found"),
				}
			},
			expectedSession: false,
		},
		{
			name:            "no_cookie",
			cookieValue:     "",
			setupSession:    func() *mockSessionService { return &mockSessionService{} },
			expectedSession: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionService := tt.setupSession()
			middleware := NewSessionMiddleware(sessionService, "locky_session")

			// Setup context with tenant
			tenant := &core.Tenant{
				ID:   "tenant-123",
				Slug: "test",
			}

			handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				session, ok := GetSession(r.Context())
				if tt.expectedSession {
					assert.True(t, ok)
					assert.NotNil(t, session)
				} else {
					assert.False(t, ok)
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/", nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  "locky_session",
					Value: tt.cookieValue,
				})
			}

			// Add tenant to context
			ctx := context.WithValue(req.Context(), ContextKeyTenant, tenant)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestGetTenant(t *testing.T) {
	tenant := &core.Tenant{
		ID:   "tenant-123",
		Slug: "test",
	}

	ctx := context.WithValue(context.Background(), ContextKeyTenant, tenant)
	result, ok := GetTenant(ctx)

	assert.True(t, ok)
	assert.Equal(t, tenant, result)
}

func TestGetTenant_NotFound(t *testing.T) {
	ctx := context.Background()
	result, ok := GetTenant(ctx)

	assert.False(t, ok)
	assert.Nil(t, result)
}

func TestGetSession(t *testing.T) {
	session := &core.Session{
		ID:       "session-123",
		TenantID: "tenant-123",
		UserID:   "user-456",
	}

	ctx := context.WithValue(context.Background(), ContextKeySession, session)
	result, ok := GetSession(ctx)

	assert.True(t, ok)
	assert.Equal(t, session, result)
}

func TestGetSession_NotFound(t *testing.T) {
	ctx := context.Background()
	result, ok := GetSession(ctx)

	assert.False(t, ok)
	assert.Nil(t, result)
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()
	writeError(rr, http.StatusBadRequest, "invalid_request", "Invalid request parameters")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "invalid_request")
	assert.Contains(t, rr.Body.String(), "Invalid request parameters")
}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	data := []byte(`{"status":"ok","version":"1.0"}`)
	writeJSON(rr, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Equal(t, data, rr.Body.Bytes())
}
