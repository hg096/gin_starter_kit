package config

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// Config 애플리케이션 전체 설정을 담는 구조체
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
	AccessSecret       []byte
	RefreshSecret      []byte
	TokenSecret        []byte
	AccessExpireMin    int
	RefreshExpireDays  int
	RefreshReuseHours  int // 리프레시 토큰 재사용 기준 시간
}

type AppConfig struct {
	ServiceName string
	Environment string
	Debug       bool
}

var (
	instance *Config
	once     sync.Once
)

// Load .env 파일을 로드하고 설정을 초기화
func Load() *Config {
	once.Do(func() {
		// .env 파일 로드
		if err := godotenv.Load(); err != nil {
			log.Println("⚠️  .env 파일을 찾을 수 없습니다. 환경변수를 사용합니다.")
		}

		instance = &Config{
			Server:   loadServerConfig(),
			Database: loadDatabaseConfig(),
			JWT:      loadJWTConfig(),
			App:      loadAppConfig(),
		}

		// 필수 값 검증
		instance.validate()
	})
	return instance
}

// Get 싱글톤 인스턴스 반환
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

	// 32바이트 검증
	if len(accessSecret) != 32 || len(refreshSecret) != 32 || len(tokenSecret) != 32 {
		log.Fatal("❌ JWT_SECRET, JWT_REFRESH_SECRET, JWT_TOKEN_SECRET는 각각 32자여야 합니다")
	}

	return JWTConfig{
		AccessSecret:      []byte(accessSecret),
		RefreshSecret:     []byte(refreshSecret),
		TokenSecret:       []byte(tokenSecret),
		AccessExpireMin:   getEnvAsInt("JWT_EXPIRES_IN", 30),
		RefreshExpireDays: getEnvAsInt("JWT_EXPIRES_RE", 7),
		RefreshReuseHours: 24, // 리프레시 토큰 24시간 이상 남으면 재사용
	}
}

func loadAppConfig() AppConfig {
	ginMode := getEnv("GIN_MODE", "debug")
	debug := ginMode == "debug"

	return AppConfig{
		ServiceName: getEnv("SERVICE_NAME", "GinStarter"),
		Environment: ginMode,
		Debug:       debug,
	}
}

// validate 필수 설정값 검증
func (c *Config) validate() {
	if c.Database.Database == "" {
		log.Fatal("❌ DB_NAME이 설정되지 않았습니다")
	}
	if c.Database.Password == "" {
		log.Println("⚠️  DB_PASS가 비어있습니다")
	}
}

// IsDevelopment 개발 환경인지 확인
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "debug"
}

// IsProduction 운영 환경인지 확인
func (c *Config) IsProduction() bool {
	return c.App.Environment == "release"
}

// GetDSN MySQL DSN 문자열 생성
func (c *Config) GetDSN() string {
	return c.Database.User + ":" + c.Database.Password +
		"@tcp(" + c.Database.Host + ":" + c.Database.Port + ")/" +
		c.Database.Database + "?charset=utf8mb4&parseTime=True&loc=Local"
}

// Helper functions
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