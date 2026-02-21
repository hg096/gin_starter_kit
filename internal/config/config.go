package config

import (
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// Config stores application settings.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	App      AppConfig
}

type ServerConfig struct {
	Port     string
	GinMode  string
	Timeout  time.Duration
	BasePath string
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type JWTConfig struct {
	AccessSecret      []byte
	RefreshSecret     []byte
	TokenSecret       []byte
	AccessExpireMin   int
	RefreshExpireDays int
	RefreshReuseHours int
	Issuer            string
	Audience          string
	Subject           string
}

type AppConfig struct {
	ServiceName        string
	Environment        string
	Debug              bool
	CORSAllowedOrigins []string
}

var (
	instance *Config
	once     sync.Once
)

// Load loads .env and initializes config.
func Load() *Config {
	once.Do(func() {
		if err := godotenv.Load(); err != nil {
			log.Println(".env file not found, using environment variables")
		}

		instance = &Config{
			Server:   loadServerConfig(),
			Database: loadDatabaseConfig(),
			JWT:      loadJWTConfig(),
			App:      loadAppConfig(),
		}

		instance.validate()
	})
	return instance
}

// Get returns singleton config instance.
func Get() *Config {
	if instance == nil {
		return Load()
	}
	return instance
}

func loadServerConfig() ServerConfig {
	port := getEnv("PORT", "8080")
	ginMode := getEnv("GIN_MODE", "debug")
	timeout := getEnvAsInt("SERVER_TIMEOUT", 30)

	return ServerConfig{
		Port:     port,
		GinMode:  ginMode,
		Timeout:  time.Duration(timeout) * time.Second,
		BasePath: "/",
	}
}

func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnv("DB_PORT", "3306"),
		User:            getEnv("DB_USER", "root"),
		Password:        getEnv("DB_PASS", ""),
		Database:        getEnv("DB_NAME", ""),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME", 5)) * time.Minute,
	}
}

func loadJWTConfig() JWTConfig {
	accessSecret := getEnv("JWT_SECRET", "")
	refreshSecret := getEnv("JWT_REFRESH_SECRET", "")
	tokenSecret := getEnv("JWT_TOKEN_SECRET", "")
	issuer := strings.TrimSpace(getEnv("JWT_ISSUER", ""))
	audience := strings.TrimSpace(getEnv("JWT_AUDIENCE", ""))
	subject := strings.TrimSpace(getEnv("JWT_SUBJECT", ""))

	if len(accessSecret) != 32 || len(refreshSecret) != 32 || len(tokenSecret) != 32 {
		log.Fatal("JWT_SECRET, JWT_REFRESH_SECRET, JWT_TOKEN_SECRET must each be 32 characters")
	}

	if issuer == "" || audience == "" || subject == "" {
		log.Fatal("JWT_ISSUER, JWT_AUDIENCE, JWT_SUBJECT are required")
	}

	return JWTConfig{
		AccessSecret:      []byte(accessSecret),
		RefreshSecret:     []byte(refreshSecret),
		TokenSecret:       []byte(tokenSecret),
		AccessExpireMin:   getEnvAsInt("JWT_EXPIRES_IN", 30),
		RefreshExpireDays: getEnvAsInt("JWT_EXPIRES_RE", 7),
		RefreshReuseHours: 24,
		Issuer:            issuer,
		Audience:          audience,
		Subject:           subject,
	}
}

func loadAppConfig() AppConfig {
	ginMode := getEnv("GIN_MODE", "debug")
	debug := ginMode == "debug"

	return AppConfig{
		ServiceName: getEnv("SERVICE_NAME", "GinStarter"),
		Environment: ginMode,
		Debug:       debug,
		CORSAllowedOrigins: getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{
			"http://localhost:8080",
			"http://127.0.0.1:8080",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}),
	}
}

// validate validates required settings.
func (c *Config) validate() {
	if c.Database.Database == "" {
		log.Fatal("DB_NAME is required")
	}
	if c.Database.Password == "" {
		log.Println("warning: DB_PASS is empty")
	}
}

// IsDevelopment returns true in debug mode.
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "debug"
}

// IsProduction returns true in release mode.
func (c *Config) IsProduction() bool {
	return c.App.Environment == "release"
}

// IsAllowedOrigin checks whether an origin is in allow-list.
func (c *Config) IsAllowedOrigin(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return false
	}

	for _, allowed := range c.App.CORSAllowedOrigins {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}

	// Developer ergonomics: in debug mode, allow same local port origins
	// (e.g. admin page and websocket served from localhost:<PORT>).
	if c != nil && c.IsDevelopment() {
		parsed, err := url.Parse(origin)
		if err == nil {
			host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
			port := strings.TrimSpace(parsed.Port())
			if port == "" {
				switch strings.ToLower(parsed.Scheme) {
				case "https":
					port = "443"
				case "http":
					port = "80"
				}
			}

			serverPort := strings.TrimSpace(c.Server.Port)
			if serverPort == "" {
				serverPort = "8080"
			}

			if port == serverPort && (host == "localhost" || host == "127.0.0.1" || host == "::1") {
				return true
			}
		}
	}

	return false
}

// GetDSN builds MySQL DSN.
func (c *Config) GetDSN() string {
	return c.Database.User + ":" + c.Database.Password +
		"@tcp(" + c.Database.Host + ":" + c.Database.Port + ")/" +
		c.Database.Database + "?charset=utf8mb4&parseTime=True&loc=Local"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	raw := getEnv(key, "")
	if raw == "" {
		return defaultValue
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}

	if len(values) == 0 {
		return defaultValue
	}

	return values
}
