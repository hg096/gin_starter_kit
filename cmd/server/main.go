package main

import (
	"context"
	"gin_starter/api/routes"
	"gin_starter/internal/config"
	"gin_starter/internal/infrastructure/database"
	"gin_starter/internal/websocket"
	"gin_starter/pkg/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "gin_starter/docs" // Swagger docs

	"github.com/gin-gonic/gin"
)

// @title Gin Starter API
// @version 2.0
// @description 개선된 Gin Starter Kit API 서버
// @BasePath /
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
func main() {
	// 설정 로드
	cfg := config.Load()

	// 로그 레벨 설정
	logger.SetLevelFromString(cfg.App.Environment)
	logger.Info("🚀 서버 시작 중... (환경: %s)", cfg.App.Environment)

	// Gin 모드 설정
	gin.SetMode(cfg.Server.GinMode)

	// 데이터베이스 연결
	db, err := database.Connect(cfg)
	if err != nil {
		logger.Fatal("데이터베이스 연결 실패: %v", err)
	}
	defer db.Close()

	// WebSocket Hub 생성 및 시작
	hub := websocket.NewHub()
	go hub.Run()
	logger.Info("WebSocket Hub 시작됨")

	// Gin 엔진 생성
	r := gin.New()

	// 라우트 설정
	routes.SetupRoutes(r, db, cfg)

	// WebSocket 라우트 설정
	websocket.SetupWebSocketRoutes(r, hub, cfg)

	// HTTP 서버 설정
	srv := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        r,
		ReadTimeout:    cfg.Server.Timeout,
		WriteTimeout:   cfg.Server.Timeout,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// 서버 시작 (고루틴)
	go func() {
		logger.Info("✅ 서버가 포트 %s에서 시작되었습니다", cfg.Server.Port)
		logger.Info("📖 Swagger 문서: http://localhost:%s/swagger/index.html", cfg.Server.Port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("서버 시작 실패: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 서버 종료 중...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("서버 강제 종료: %v", err)
	}

	logger.Info("👋 서버가 정상적으로 종료되었습니다")
}