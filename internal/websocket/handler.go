package websocket

import (
	"gin_starter/internal/config"
	"gin_starter/internal/middleware"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// CORS 설정 - 실제 운영 환경에서는 특정 도메인만 허용해야 함
		return true
	},
}

// Handler WebSocket 핸들러
type Handler struct {
	hub *Hub
	cfg *config.Config
}

// NewHandler WebSocket 핸들러 생성
func NewHandler(hub *Hub, cfg *config.Config) *Handler {
	return &Handler{
		hub: hub,
		cfg: cfg,
	}
}

// HandleChat 채팅 WebSocket 연결
// @Summary      채팅 WebSocket
// @Description  실시간 채팅을 위한 WebSocket 연결
// @Tags         websocket
// @Param        room_id query string true "방 ID"
// @Success      101
// @Security     BearerAuth
// @Router       /ws/chat [get]
func (h *Handler) HandleChat(c *gin.Context) {
	// 인증 확인 (미들웨어에서 설정한 값 가져오기)
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "인증이 필요합니다")
		return
	}

	// 방 ID 파라미터
	roomID := c.Query("room_id")
	if roomID == "" {
		response.BadRequest(c, "room_id는 필수입니다")
		return
	}

	// WebSocket 업그레이드
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("WebSocket 업그레이드 실패: %v", err)
		return
	}

	// 클라이언트 생성 및 등록
	client := NewClient(h.hub, conn, userID.(string), roomID)
	h.hub.register <- client

	// 고루틴으로 읽기/쓰기 처리
	go client.WritePump()
	go client.ReadPump()
}

// GetRoomInfo 방 정보 조회
// @Summary      방 정보 조회
// @Description  특정 방의 접속자 목록과 정보를 조회합니다
// @Tags         websocket
// @Accept       json
// @Produce      json
// @Param        room_id path string true "방 ID"
// @Success      200 {object} response.Response
// @Security     BearerAuth
// @Router       /api/ws/room/{room_id} [get]
func (h *Handler) GetRoomInfo(c *gin.Context) {
	roomID := c.Param("room_id")
	if roomID == "" {
		response.BadRequest(c, "room_id는 필수입니다")
		return
	}

	clients := h.hub.GetRoomClients(roomID)

	response.Success(c, gin.H{
		"room_id":      roomID,
		"client_count": len(clients),
		"clients":      clients,
	})
}

// GetStats WebSocket 통계
// @Summary      WebSocket 통계
// @Description  전체 방 개수와 접속자 수를 조회합니다
// @Tags         websocket
// @Accept       json
// @Produce      json
// @Success      200 {object} response.Response
// @Security     BearerAuth
// @Router       /api/ws/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	response.Success(c, gin.H{
		"room_count":   h.hub.GetRoomCount(),
		"client_count": h.hub.GetClientCount(),
	})
}

// SetupWebSocketRoutes WebSocket 라우트 설정
func SetupWebSocketRoutes(r *gin.Engine, hub *Hub, cfg *config.Config) {
	handler := NewHandler(hub, cfg)

	// WebSocket 엔드포인트 (인증 필요)
	ws := r.Group("/ws")
	ws.Use(middleware.AuthMiddleware(cfg))
	{
		ws.GET("/chat", handler.HandleChat)
	}

	// WebSocket API 엔드포인트 (인증 필요)
	api := r.Group("/api/ws")
	api.Use(middleware.AuthMiddleware(cfg))
	{
		api.GET("/room/:room_id", handler.GetRoomInfo)
		api.GET("/stats", handler.GetStats)
	}
}