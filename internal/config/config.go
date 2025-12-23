package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Security SecurityConfig
}

type ServerConfig struct {
	Port             string
	Host             string
	Environment      string
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	CORSAllowOrigins []string
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxConnections  int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type JWTConfig struct {
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	PrivateKey           *rsa.PrivateKey
	PublicKey            *rsa.PublicKey
	Issuer               string
}

type SecurityConfig struct {
	BCryptCost          int
	RateLimitPerSecond  int
	MaxFailedAttempts   int
	PasswordMinLength   int
	RequireUppercase    bool
	RequireLowercase    bool
	RequireNumbers      bool
	RequireSpecialChars bool
}

func Load() *Config {
	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "localhost"),
			Environment:  getEnv("APP_ENV", "development"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "banking_user"),
			Password:        getEnv("DB_PASSWORD", "banking_password"),
			Name:            getEnv("DB_NAME", "banking_db"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxConnections:  getIntEnv("DB_MAX_CONNECTIONS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", time.Hour),
		},
		Security: SecurityConfig{
			BCryptCost:          getIntEnv("BCRYPT_COST", 12),
			RateLimitPerSecond:  getIntEnv("RATE_LIMIT_PER_SECOND", 5),
			MaxFailedAttempts:   getIntEnv("MAX_FAILED_ATTEMPTS", 3),
			PasswordMinLength:   getIntEnv("PASSWORD_MIN_LENGTH", 12),
			RequireUppercase:    getBoolEnv("PASSWORD_REQUIRE_UPPERCASE", true),
			RequireLowercase:    getBoolEnv("PASSWORD_REQUIRE_LOWERCASE", true),
			RequireNumbers:      getBoolEnv("PASSWORD_REQUIRE_NUMBERS", true),
			RequireSpecialChars: getBoolEnv("PASSWORD_REQUIRE_SPECIAL", true),
		},
		JWT: JWTConfig{
			AccessTokenDuration:  getDurationEnv("JWT_ACCESS_TOKEN_DURATION", 24*time.Hour),
			RefreshTokenDuration: getDurationEnv("JWT_REFRESH_TOKEN_DURATION", 7*24*time.Hour),
			Issuer:               getEnv("JWT_ISSUER", "banking-api"),
		},
	}

	config.Server.CORSAllowOrigins = config.loadCORSAllowOrigins()

	var loadJWTKeysErr error
	config.JWT.PrivateKey, config.JWT.PublicKey, loadJWTKeysErr = config.loadJWTKeys()
	if loadJWTKeysErr != nil {
		log.Fatal("Failed to load RSA keys:", loadJWTKeysErr)
	}

	return config
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

func (c *Config) IsTesting() bool {
	return c.Server.Environment == "testing"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// loadJWTKeys loads RSA keys for JWT signing and verification
// Priority order:
// 1. If JWT_PRIVATE_KEY and JWT_PUBLIC_KEY env vars are set, use them (works in all environments)
// 2. If production and env vars missing, fail with error (production requires explicit keys)
// 3. If development/testing and env vars missing, generate new keypair (dev convenience)
func (c *Config) loadJWTKeys() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKeyB64 := os.Getenv("JWT_PRIVATE_KEY")
	publicKeyB64 := os.Getenv("JWT_PUBLIC_KEY")

	if privateKeyB64 != "" && publicKeyB64 != "" {
		log.Println("Loading RSA keypair from environment variables")
		return c.loadKeysFromEnvVars(privateKeyB64, publicKeyB64)
	}

	if c.IsProduction() {
		return nil, nil, fmt.Errorf("JWT_PRIVATE_KEY and JWT_PUBLIC_KEY environment variables must be set in production environments")
	}

	log.Println("Development environment: generating new RSA keypair for JWT (consider setting JWT_PRIVATE_KEY and JWT_PUBLIC_KEY env vars to persist keys across restarts)")
	return GenerateRSAKeyPair()
}

// loadKeysFromEnvVars loads RSA keys from base64-encoded environment variables
func (c *Config) loadKeysFromEnvVars(privateKeyB64, publicKeyB64 string) (*rsa.PrivateKey, *rsa.PublicKey, error) {

	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode JWT_PRIVATE_KEY: %w", err)
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode JWT_PUBLIC_KEY: %w", err)
	}

	privateKey, err := loadRSAPrivateKey(privateKeyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	publicKey, err := loadRSAPublicKey(publicKeyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return privateKey, publicKey, nil
}

// loadCORSAllowOrigins retrieves CORS allowed origins from environment or returns default
func (c *Config) loadCORSAllowOrigins() []string {
	corsOrigins := os.Getenv("CORS_ALLOW_ORIGINS")

	if corsOrigins == "" {
		if c.IsProduction() {
			log.Println("WARNING: CORS_ALLOW_ORIGINS not set in production environment, defaulting to '*' (all origins). Consider setting specific origins for security.")
		} else {
			log.Println("INFO: CORS_ALLOW_ORIGINS not set, defaulting to '*' (all origins)")
		}
		return []string{"*"}
	}

	// Split by comma and trim whitespace
	origins := strings.Split(corsOrigins, ",")
	for i, origin := range origins {
		origins[i] = strings.TrimSpace(origin)
	}

	log.Printf("CORS allowed origins configured: %v", origins)
	return origins
}

// GenerateRSAKeyPair generates a new RSA key pair
func GenerateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	return privateKey, &privateKey.PublicKey, nil
}

// loadRSAPrivateKey loads an RSA private key from PEM format
func loadRSAPrivateKey(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Fallback: PKCS8 format support for compatibility with various key generation tools
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		privateKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an RSA private key")
		}

		return privateKey, nil
	}

	return privateKey, nil
}

// loadRSAPublicKey loads an RSA public key from PEM format
func loadRSAPublicKey(pemData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaPublicKey, nil
}
