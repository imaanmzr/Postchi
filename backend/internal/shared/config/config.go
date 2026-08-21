package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/imaanmzr/postchi/backend/internal/shared/emailpolicy"
)

// Pinned versions: Go 1.26.3, chi v5.2.1, pgx v5.7.4
type Config struct {
	Environment      string
	HTTPPort         string
	DatabaseURL      string
	JWTSecret        string
	JWTIssuer        string
	EncryptionKey    string
	CORSOrigins      []string
	ShutdownTimeout  time.Duration
	RequestTimeout   time.Duration
	MaxResponseBytes int64
	MigrationsPath   string
	AutoMigrate      bool
	DBReadyTimeout   time.Duration
	StaticFilesPath  string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	AppPublicURL     string
	SMTPHost         string
	SMTPPort         int
	SMTPUser         string
	SMTPPass         string
	SMTPFrom         string
	InviteTTL                    time.Duration
	PasswordResetTTL             time.Duration
	RegistrationAllowedDomains   []string
}

func Load() (*Config, error) {
	shutdown, _ := time.ParseDuration(getEnv("SHUTDOWN_TIMEOUT", "10s"))
	dbReady, _ := time.ParseDuration(getEnv("DB_READY_TIMEOUT", "60s"))
	reqTimeoutSec, _ := strconv.Atoi(getEnv("REQUEST_TIMEOUT_SECONDS", "30"))
	maxBytes, _ := strconv.ParseInt(getEnv("MAX_RESPONSE_BYTES", "10485760"), 10, 64)
	inviteHours, _ := strconv.Atoi(getEnv("INVITE_TTL_HOURS", "168"))
	passwordResetHours, _ := strconv.Atoi(getEnv("PASSWORD_RESET_TTL_HOURS", "1"))
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))

	cfg := &Config{
		Environment:      getEnv("ENVIRONMENT", "development"),
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://postchi:postchi@localhost:5432/postchi?sslmode=disable"),
		JWTSecret:        getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		JWTIssuer:        getEnv("JWT_ISSUER", "postchi"),
		EncryptionKey:    getEnv("ENCRYPTION_KEY", "postchi-dev-encryption-key-32b!!"),
		CORSOrigins:      parseCORSOrigins(getEnv("CORS_ORIGINS", "http://localhost:3000")),
		ShutdownTimeout:  shutdown,
		RequestTimeout:   time.Duration(reqTimeoutSec) * time.Second,
		MaxResponseBytes: maxBytes,
		MigrationsPath:   getEnv("MIGRATIONS_PATH", "file://migrations"),
		AutoMigrate:      getEnvBool("AUTO_MIGRATE", true),
		DBReadyTimeout:   dbReady,
		StaticFilesPath:  getEnv("STATIC_FILES_PATH", ""),
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  7 * 24 * time.Hour,
		AppPublicURL:     getEnv("APP_PUBLIC_URL", "http://localhost:3000"),
		SMTPHost:         getEnv("SMTP_HOST", ""),
		SMTPPort:         smtpPort,
		SMTPUser:         getEnv("SMTP_USER", ""),
		SMTPPass:         getEnv("SMTP_PASS", ""),
		SMTPFrom:         getEnv("SMTP_FROM", "postchi@localhost"),
		InviteTTL:                  time.Duration(inviteHours) * time.Hour,
		PasswordResetTTL:             time.Duration(passwordResetHours) * time.Hour,
		RegistrationAllowedDomains: parseRegistrationDomains(getEnv("REGISTRATION_ALLOWED_EMAIL_DOMAINS", "")),
	}
	return cfg, nil
}

func parseRegistrationDomains(raw string) []string {
	parts := strings.Split(raw, ",")
	var out []string
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (c *Config) RegistrationDomainAllowed(email string) bool {
	return emailpolicy.Allowed(email, c.RegistrationAllowedDomains)
}

func parseCORSOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"http://localhost:3000"}
	}
	return out
}

func (c *Config) SMTPConfigured() bool {
	return c.SMTPHost != ""
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
