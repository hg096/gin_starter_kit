package routes

import (
	"encoding/json"
	"gin_starter/internal/config"
	"gin_starter/internal/infrastructure/database"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func testRouteConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:      []byte("01234567890123456789012345678901"),
			RefreshSecret:     []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			TokenSecret:       []byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
			AccessExpireMin:   30,
			RefreshExpireDays: 7,
			RefreshReuseHours: 24,
		},
		App: config.AppConfig{
			ServiceName:        "test-service",
			Environment:        "debug",
			CORSAllowedOrigins: []string{"http://localhost:3000"},
		},
	}
}

func TestHealthRoutes_LiveAndReadyCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, nil, testRouteConfig())

	liveReq := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	liveRec := httptest.NewRecorder()
	r.ServeHTTP(liveRec, liveReq)

	if liveRec.Code != http.StatusOK {
		t.Fatalf("expected /health/live to return 200, got %d", liveRec.Code)
	}

	readyReq := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	readyRec := httptest.NewRecorder()
	r.ServeHTTP(readyRec, readyReq)

	if readyRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected /health/ready to return 503 without DB, got %d", readyRec.Code)
	}

	legacyReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	legacyRec := httptest.NewRecorder()
	r.ServeHTTP(legacyRec, legacyReq)

	if legacyRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected /health to mirror readiness (503), got %d", legacyRec.Code)
	}

	var readyBody map[string]string
	if err := json.Unmarshal(readyRec.Body.Bytes(), &readyBody); err != nil {
		t.Fatalf("failed to decode readiness body: %v", err)
	}
	if readyBody["database"] != "disconnected" {
		t.Fatalf("expected disconnected database state, got %q", readyBody["database"])
	}
}

func TestHealthReady_Returns200WhenDBIsHealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	defer sqlDB.Close()

	mock.ExpectPing()

	r := gin.New()
	SetupRoutes(r, &database.DB{DB: sqlDB}, testRouteConfig())

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected /health/ready to return 200 when DB ping succeeds, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode readiness body: %v", err)
	}
	if body["database"] != "connected" {
		t.Fatalf("expected connected database state, got %q", body["database"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
