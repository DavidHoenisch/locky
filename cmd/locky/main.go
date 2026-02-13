package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/locky/auth/core"
	"github.com/locky/auth/crypto"
	authhttp "github.com/locky/auth/http"
	"github.com/locky/auth/oauth"
	"github.com/locky/auth/rbac"
	"github.com/locky/auth/sessions"
	"github.com/locky/auth/store"
	"github.com/locky/auth/tenant"
	"github.com/locky/auth/tokens"
)

func main() {
	var (
		databaseURL   = flag.String("database-url", getEnv("DATABASE_URL", "postgres://localhost/locky?sslmode=disable"), "Database URL")
		adminAPIKey   = flag.String("admin-api-key", getEnv("ADMIN_API_KEY", ""), "Admin API key for bootstrap")
		baseDomain    = flag.String("base-domain", getEnv("BASE_DOMAIN", "auth.example.com"), "Base domain for tenant subdomains")
		httpAddr      = flag.String("http-addr", getEnv("HTTP_ADDR", ":8080"), "HTTP server address")
		enableUI      = flag.Bool("enable-ui", getEnvBool("ENABLE_UI", true), "Enable hosted UI")
		enableAdminUI = flag.Bool("enable-admin-ui", getEnvBool("ENABLE_ADMIN_UI", false), "Enable admin UI")
		adminUIUser   = flag.String("admin-ui-username", getEnv("ADMIN_UI_USERNAME", "admin"), "Admin UI username")
		adminUIPass   = flag.String("admin-ui-password", getEnv("ADMIN_UI_PASSWORD", "admin123"), "Admin UI password")
		autoMigrate   = flag.Bool("auto-migrate", getEnvBool("AUTO_MIGRATE", true), "Auto-run database migrations")
	)
	flag.Parse()

	// Initialize store
	log.Println("Connecting to database...")
	gormStore, err := store.New(*databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate
	if *autoMigrate {
		log.Println("Running database migrations...")
		if err := gormStore.AutoMigrate(); err != nil {
			log.Fatalf("Failed to migrate database: %v", err)
		}
	}

	// Create core configuration
	cfg := core.Config{
		DatabaseURL:           *databaseURL,
		AdminAPIKey:           *adminAPIKey,
		BaseDomain:            *baseDomain,
		SessionCookieName:     "locky_session",
		SessionCookieSecure:   true,
		SessionCookieSameSite: "Lax",
		AccessTokenTTL:        15 * time.Minute,
		RefreshTokenTTL:       14 * 24 * time.Hour,
		SessionTTL:            30 * 24 * time.Hour,
		MaxLoginAttempts:      5,
		PasswordMinLength:     8,
		EnableHostedUI:        *enableUI,
		EnableAdminUI:         *enableAdminUI,
		AdminUIUsername:       *adminUIUser,
		AdminUIPassword:       *adminUIPass,
	}

	// Initialize services
	clock := core.RealClock{}

	// Initialize crypto services
	jwtManager := crypto.NewJWTManager(gormStore.SigningKeys())
	keyManager := crypto.NewKeyManager(gormStore.SigningKeys(), nil) // TODO: provide master key

	// Initialize tenant resolver
	tenantResolver := tenant.NewHostResolver(gormStore.Domains(), gormStore.Tenants(), *baseDomain)

	// Initialize RBAC
	rbacService, err := rbac.NewService(gormStore.DB())
	if err != nil {
		log.Fatalf("Failed to initialize RBAC: %v", err)
	}

	// Initialize token service
	tokenService := tokens.NewService(
		gormStore.SigningKeys(),
		gormStore.OAuthCodes(),
		gormStore.RefreshTokens(),
		gormStore.Sessions(),
		jwtManager,
		clock,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
	)

	// Initialize session service
	sessionService := sessions.NewService(gormStore.Sessions(), clock, cfg.SessionTTL)

	// Initialize OAuth service
	oauthService := oauth.NewService(
		gormStore.Clients(),
		gormStore.Users(),
		gormStore.OAuthCodes(),
		gormStore.RefreshTokens(),
		tokenService,
		sessionService,
		tenantResolver,
		nil, // audit sink
		clock,
		10*time.Minute,
	)

	// Create the core
	coreInstance, err := core.NewCore(cfg, gormStore, rbacService, nil)
	if err != nil {
		log.Fatalf("Failed to create core: %v", err)
	}

	// Set service implementations
	coreInstance.KeyManager = keyManager
	coreInstance.TenantResolver = tenantResolver
	coreInstance.TokenService = tokenService
	coreInstance.SessionService = sessionService
	coreInstance.OAuthService = oauthService

	// Create admin API key if provided
	if *adminAPIKey != "" {
		// TODO: Bootstrap admin key
		log.Println("Admin API key configured")
	}

	// Create HTTP server
	log.Printf("Starting HTTP server on %s...", *httpAddr)
	server := authhttp.NewServer(coreInstance, cfg)

	// Start server
	if err := http.ListenAndServe(*httpAddr, server); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}
