# Locky Test Suite

This document describes the comprehensive test suite for Locky.

## Test Structure

```
auth/
├── core/
│   └── models_test.go          # Domain model validation tests
├── crypto/
│   └── crypto_test.go          # Password hashing and JWT tests
├── tenant/
│   └── resolver_test.go        # Tenant resolution tests
├── sessions/
│   └── service_test.go         # Session management tests
├── http/
│   └── middleware_test.go      # HTTP middleware tests
└── store/
    └── store_test.go           # Database persistence tests
```

## Running Tests

### Run All Tests

```bash
cd auth
go test ./...
```

### Run Tests for Specific Package

```bash
# Core domain tests
go test ./core/...

# Crypto tests
go test ./crypto/...

# Tenant resolver tests
go test ./tenant/...

# Session service tests
go test ./sessions/...

# HTTP middleware tests
go test ./http/...

# Store tests (uses SQLite)
go test ./store/...
```

### Run with Verbose Output

```bash
go test -v ./...
```

### Run with Coverage

```bash
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Categories

### 1. Unit Tests (Fast, No External Dependencies)

These tests are fast and don't require external services:

- **core/models_test.go**: Validates domain models and their behavior
- **crypto/crypto_test.go**: Tests password hashing (Argon2id) and JWT operations
- **tenant/resolver_test.go**: Tests host-based tenant resolution logic
- **sessions/service_test.go**: Tests session lifecycle management
- **http/middleware_test.go**: Tests HTTP middleware (CORS, auth, etc.)

Run with:
```bash
go test ./core/... ./crypto/... ./tenant/... ./sessions/... ./http/...
```

### 2. Integration Tests (Requires SQLite)

These tests use SQLite for database testing:

- **store/store_test.go**: Tests all database operations with real SQL

Run with:
```bash
go test ./store/...
```

## Test Coverage by Component

### Core Domain Models (models_test.go)

**Coverage Areas:**
- Tenant model validation
- User model validation (including optional fields)
- Session model validation
- Client/OAuth app validation
- OAuth authorization code validation
- Refresh token validation
- Audit event validation
- Policy and RBAC tuple validation
- Token claims validation
- OAuth/OIDC request/response validation

**Key Tests:**
- Model field presence and types
- Status field validation (active/disabled/suspended)
- Optional pointer fields (DisplayName, VerifiedAt, etc.)
- JSON tag validation
- Time field handling

### Cryptographic Services (crypto_test.go)

**Coverage Areas:**
- Password hashing with Argon2id
- Password verification
- JWT signing and verification (mocked)
- Key generation and management
- Private key encryption/decryption
- String hashing utilities

**Key Tests:**
- Argon2id parameters (time, memory, threads)
- Salt uniqueness (different hashes for same password)
- Constant-time comparison
- Invalid hash handling
- JWT token format validation
- Key rotation support

### Tenant Resolution (resolver_test.go)

**Coverage Areas:**
- Host-based tenant resolution
- Custom domain resolution
- Subdomain extraction
- Case insensitivity
- Port stripping
- Scheme handling (http/https)

**Key Tests:**
- Verified vs unverified custom domains
- Subdomain parsing (e.g., `acme.auth.example.com`)
- Multi-level subdomains
- Mixed case handling
- Invalid host handling

### Session Management (sessions/service_test.go)

**Coverage Areas:**
- Session creation
- Session validation
- Session revocation
- Session expiration
- Last seen updates
- Multi-tenant isolation

**Key Tests:**
- Valid session lifecycle
- Revoked session rejection
- Expired session cleanup
- Cross-tenant access prevention
- Concurrent session handling

### HTTP Middleware (middleware_test.go)

**Coverage Areas:**
- Tenant resolution middleware
- Admin API key validation
- Session cookie validation
- CORS handling
- Preflight request handling
- Error response formatting

**Key Tests:**
- Tenant context injection
- Missing/invalid API keys
- CORS origin validation
- Session cookie parsing
- Error JSON formatting

### Store Layer (store_test.go)

**Coverage Areas:**
- All CRUD operations for each entity
- Transaction handling
- Constraint validation
- Foreign key relationships
- Cursor-based pagination
- Soft deletion (revocation)

**Key Tests:**
- Tenant CRUD operations
- User with password hashing
- Session lifecycle
- OAuth client management
- Custom domain verification
- Refresh token rotation
- Audit event filtering
- Cleanup of expired records

## Mock Implementations

The test suite uses comprehensive mocks for external dependencies:

### Mock Store Implementations

Each store interface has a corresponding mock:

```go
type mockTenantStore struct { ... }
type mockUserStore struct { ... }
type mockSessionStore struct { ... }
type mockDomainStore struct { ... }
// etc.
```

### Mock Services

```go
type mockTenantResolver struct { ... }
type mockSessionService struct { ... }
type mockJWTManager struct { ... }
```

These mocks are defined in test files alongside the tests that use them.

## Writing New Tests

### Test Naming Conventions

- Use descriptive names: `Test<Service>_<Method>_<Scenario>`
- Examples:
  - `TestService_Create_ValidSession`
  - `TestService_Validate_ExpiredSession`
  - `TestPasswordHasher_DifferentHashes`

### Test Structure

```go
func Test<Component>_<Scenario>(t *testing.T) {
    // Setup
    service, mocks := setup()
    
    // Execute
    result, err := service.Method()
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Table-Driven Tests

For testing multiple scenarios:

```go
func TestService_MultipleScenarios(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid", "input1", "output1", false},
        {"invalid", "input2", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := service.Method(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Install dependencies
        run: cd auth && go mod download
      
      - name: Run tests
        run: cd auth && go test -v -race ./...
      
      - name: Generate coverage
        run: cd auth && go test -coverprofile=coverage.out ./...
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./auth/coverage.out
```

## Known Limitations

1. **Tokens Service Tests**: Currently use mock JWT manager. Full JWT testing requires integration setup.
2. **OAuth Service Tests**: Not yet implemented (would require comprehensive OAuth flow testing)
3. **RBAC Tests**: Basic Casbin integration exists but comprehensive policy testing is needed
4. **HTTP Handler Tests**: Admin handlers need more comprehensive testing

## Future Test Additions

- [ ] OAuth2 flow end-to-end tests
- [ ] RBAC policy enforcement tests
- [ ] Rate limiting middleware tests
- [ ] Integration tests with real PostgreSQL
- [ ] Load tests for token issuance
- [ ] Security tests (SQL injection, XSS, etc.)
- [ ] UI component tests (Svelte)