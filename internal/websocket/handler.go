package websocket

import (
	"gin_starter/internal/config"
	"gin_starter/pkg/db/database"
	appErrors "gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"gin_starter/pkg/middleware"
	"gin_starter/pkg/response"
	"gin_starter/pkg/utils"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var chatRoomKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,79}$`)

// Handler handles websocket and admin-chat API endpoints.
type Handler struct {
	hub      *Hub
	cfg      *config.Config
	chat     ChatRepository
	initOnce sync.Once
	initErr  error
}

func NewHandler(hub *Hub, cfg *config.Config, chatRepo ChatRepository) *Handler {
	return &Handler{
		hub:  hub,
		cfg:  cfg,
		chat: chatRepo,
	}
}

// HandleChat chat websocket endpoint.
// @Summary      Chat websocket
// @Description  WebSocket endpoint for room-based chat
// @Tags         websocket
// @Param        room_id query string true "room id"
// @Success      101
// @Security     BearerAuth
// @Router       /ws/chat [get]
func (h *Handler) HandleChat(c *gin.Context) {
	userID := c.GetString("user_id")
	if strings.TrimSpace(userID) == "" {
		response.Unauthorized(c, "authentication required")
		return
	}

	roomID := strings.TrimSpace(c.Query("room_id"))
	if roomID == "" {
		response.BadRequest(c, "room_id is required")
		return
	}

	h.upgradeChat(c, userID, roomID)
}

// HandleAdminChat admin-only websocket endpoint.
func (h *Handler) HandleAdminChat(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	if userID == "" {
		response.Unauthorized(c, "authentication required")
		return
	}

	if h.chat == nil {
		response.FromError(c, appErrors.New("DATABASE_ERROR", "chat repository is not available"))
		return
	}

	roomKey := strings.TrimSpace(c.DefaultQuery("room_key", AdminLoungeRoomKey))
	if !isValidChatRoomKey(roomKey) {
		response.BadRequest(c, "invalid room_key")
		return
	}

	allowed, err := h.canAccessRoom(userID, roomKey)
	if err != nil {
		response.FromError(c, err)
		return
	}
	if !allowed {
		response.Forbidden(c, "chat room access denied")
		return
	}

	h.upgradeChat(c, userID, roomKey)
}

// HandleAdminChatNotify pushes unread-count events for current user.
func (h *Handler) HandleAdminChatNotify(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString("user_id"))
	if userID == "" {
		response.Unauthorized(c, "authentication required")
		return
	}
	h.upgradeChat(c, userID, buildNotifyRoomKey(userID))
}

// ListChatRooms returns available rooms for current admin.
func (h *Handler) ListChatRooms(c *gin.Context) {
	if h.chat == nil {
		response.FromError(c, appErrors.New("DATABASE_ERROR", "chat repository is not available"))
		return
	}

	userID := strings.TrimSpace(c.GetString("user_id"))
	if userID == "" {
		response.FromError(c, appErrors.New("UNAUTHORIZED", "authentication context missing"))
		return
	}
	if err := h.ensureChatInitialized(); err != nil {
		response.FromError(c, err)
		return
	}
	if err := h.chat.EnsureRoomMember(AdminLoungeRoomKey, userID); err != nil {
		logger.Warn("skip admin-lounge membership sync for user=%s: %v", userID, err)
	}

	rooms, err := h.chat.ListRoomsForUser(userID)
	if err != nil {
		response.FromError(c, err)
		return
	}

	for i := range rooms {
		rooms[i].OnlineCount = len(h.hub.GetRoomClients(rooms[i].RoomKey))
	}

	response.Success(c, gin.H{"rooms": rooms})
}

// ListChatUsers returns admin users available for group/direct chat.
func (h *Handler) ListChatUsers(c *gin.Context) {
	if h.chat == nil {
		response.FromError(c, appErrors.New("DATABASE_ERROR", "chat repository is not available"))
		return
	}

	if err := h.ensureChatInitialized(); err != nil {
		response.FromError(c, err)
		return
	}
	users, err := h.chat.ListAdminUsers()
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"users": users})
}

// CreateGroupRoom creates an admin group room.
func (h *Handler) CreateGroupRoom(c *gin.Context) {
	if h.chat == nil {
		response.FromError(c, appErrors.New("DATABASE_ERROR", "chat repository is not available"))
		return
	}

	userID := strings.TrimSpace(c.GetString("user_id"))
	var req ChatCreateGroupRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		response.FromError(c, appErrors.New("UNAUTHORIZED", "authentication context missing"))
		return
	}
	if err := h.ensureChatInitialized(); err != nil {
		response.FromError(c, err)
		return
	}

	roomName := strings.TrimSpace(req.Name)
	if roomName == "" {
		response.FromError(c, appErrors.New("BAD_REQUEST", "group room name is required"))
		return
	}
	if len(roomName) > 120 {
		response.FromError(c, appErrors.New("BAD_REQUEST", "group room name must be <= 120 chars"))
		return
	}

	members := normalizeMemberIDs(userID, req.MemberIDs)
	if len(members) < 2 {
		response.FromError(c, appErrors.New("BAD_REQUEST", "group room requires at least two members"))
		return
	}

	roomKey, err := buildGroupRoomKey()
	if err != nil {
		response.FromError(c, appErrors.Wrap(err, "INTERNAL_ERROR", "failed to generate room key"))
		return
	}

	room, err := h.chat.CreateGroupRoom(userID, roomKey, roomName, members)
	if err != nil {
		response.FromError(c, err)
		return
	}
	room.OnlineCount = len(h.hub.GetRoomClients(room.RoomKey))

	response.Created(c, gin.H{
		"message": "group room created",
		"room":    room,
	})
}

// CreateDirectRoom creates or returns 1:1 room with target admin.
func (h *Handler) CreateDirectRoom(c *gin.Context) {
	if h.chat == nil {
		response.FromError(c, appErrors.New("DATABASE_ERROR", "chat repository is not available"))
		return
	}

	userID := strings.TrimSpace(c.GetString("user_id"))
	targetID := strings.TrimSpace(c.Param("target_id"))
	if userID == "" {
		response.FromError(c, appErrors.New("UNAUTHORIZED", "authentication context missing"))
		return
	}
	if targetID == "" {
		response.FromError(c, appErrors.New("BAD_REQUEST", "target_id is required"))
		return
	}
	if strings.EqualFold(userID, targetID) {
		response.FromError(c, appErrors.New("BAD_REQUEST", "cannot create direct room with yourself"))
		return
	}
	if err := h.ensureChatInitialized(); err != nil {
		response.FromError(c, err)
		return
	}

	roomKey := buildDirectRoomKey(userID, targetID)
	roomName := "1:1 " + userID + " ↔ " + targetID
	room, err := h.chat.GetOrCreateDirectRoom(userID, roomKey, roomName, targetID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	room.OnlineCount = len(h.hub.GetRoomClients(room.RoomKey))

	response.Success(c, gin.H{"room": room})
}

// ListChatMessages returns room messages (supports before_id pagination).
func (h *Handler) ListChatMessages(c *gin.Context) {
	if h.chat == nil {
		response.FromError(c, appErrors.New("DATABASE_ERROR", "chat repository is not available"))
		return
	}

	userID := strings.TrimSpace(c.GetString("user_id"))
	roomKey := strings.TrimSpace(c.Param("room_key"))
	if !isValidChatRoomKey(roomKey) {
		response.BadRequest(c, "invalid room_key")
		return
	}

	limit := chatQueryNumeric[int](c, "limit", "50", 50)
	beforeID := chatQueryNumeric[int64](c, "before_id", "0", 0)

	allowed, err := h.canAccessRoom(userID, roomKey)
	if err != nil {
		response.FromError(c, err)
		return
	}
	if !allowed {
		response.FromError(c, appErrors.New("FORBIDDEN", "chat room access denied"))
		return
	}
	messages, err := h.chat.ListMessages(roomKey, limit, beforeID)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"messages": messages})
}

func chatQueryNumeric[T utils.Numeric](c *gin.Context, key string, defaultRaw string, fallback T) T {
	value, err := utils.StringToNumeric[T](strings.TrimSpace(c.DefaultQuery(key, defaultRaw)))
	if err != nil {
		return fallback
	}
	return value
}

// CreateChatMessage creates a chat message or reply(comment) in room.
func (h *Handler) CreateChatMessage(c *gin.Context) {
	if h.chat == nil {
		response.FromError(c, appErrors.New("DATABASE_ERROR", "chat repository is not available"))
		return
	}

	userID := strings.TrimSpace(c.GetString("user_id"))
	roomKey := strings.TrimSpace(c.Param("room_key"))
	if !isValidChatRoomKey(roomKey) {
		response.BadRequest(c, "invalid room_key")
		return
	}

	var req ChatCreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		response.FromError(c, appErrors.New("BAD_REQUEST", "message content is required"))
		return
	}
	if len(content) > 2000 {
		response.FromError(c, appErrors.New("BAD_REQUEST", "message content must be <= 2000 chars"))
		return
	}

	allowed, err := h.canAccessRoom(userID, roomKey)
	if err != nil {
		response.FromError(c, err)
		return
	}
	if !allowed {
		response.FromError(c, appErrors.New("FORBIDDEN", "chat room access denied"))
		return
	}

	message, err := h.chat.CreateMessage(roomKey, userID, content, req.ParentID)
	if err != nil {
		response.FromError(c, err)
		return
	}

	h.hub.Publish(&Message{
		Type:   "chat_message",
		Room:   roomKey,
		UserID: userID,
		Content: gin.H{
			"id":          message.ID,
			"room_key":    message.RoomKey,
			"parent_id":   message.ParentID,
			"sender_id":   message.SenderID,
			"sender_name": message.SenderName,
			"content":     message.Content,
			"created_at":  message.CreatedAt,
		},
	})
	h.publishRoomUnreadSnapshots(roomKey)

	response.Created(c, gin.H{"message": message})
}

// MarkChatRoomRead marks room messages as read for current user.
func (h *Handler) MarkChatRoomRead(c *gin.Context) {
	if h.chat == nil {
		response.FromError(c, appErrors.New("DATABASE_ERROR", "chat repository is not available"))
		return
	}

	userID := strings.TrimSpace(c.GetString("user_id"))
	roomKey := strings.TrimSpace(c.Param("room_key"))
	if !isValidChatRoomKey(roomKey) {
		response.BadRequest(c, "invalid room_key")
		return
	}

	var req ChatMarkReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Empty body is allowed.
		req.LastMessageID = 0
	}

	allowed, err := h.canAccessRoom(userID, roomKey)
	if err != nil {
		response.FromError(c, err)
		return
	}
	if !allowed {
		response.FromError(c, appErrors.New("FORBIDDEN", "chat room access denied"))
		return
	}

	if err := h.chat.MarkRoomRead(roomKey, userID, req.LastMessageID); err != nil {
		response.FromError(c, err)
		return
	}
	h.publishRoomUnreadSnapshots(roomKey)
	response.Success(c, gin.H{"message": "room marked as read"})
}

func (h *Handler) upgradeChat(c *gin.Context, userID string, roomID string) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin == "" {
				return true
			}
			if h.cfg != nil && h.cfg.IsAllowedOrigin(origin) {
				return true
			}
			logger.Warn("WebSocket origin blocked: %s", origin)
			return false
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("websocket upgrade failed: %v", err)
		return
	}

	client := NewClient(h.hub, conn, userID, roomID)
	h.hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

// GetRoomInfo returns room stats.
// @Summary      Get websocket room info
// @Description  Returns clients connected to a specific room
// @Tags         websocket
// @Accept       json
// @Produce      json
// @Param        room_id path string true "room id"
// @Success      200 {object} response.Response
// @Security     BearerAuth
// @Router       /api/ws/room/{room_id} [get]
func (h *Handler) GetRoomInfo(c *gin.Context) {
	roomID := strings.TrimSpace(c.Param("room_id"))
	if roomID == "" {
		response.BadRequest(c, "room_id is required")
		return
	}

	if h.chat != nil && isManagedAdminRoom(roomID) {
		userID := strings.TrimSpace(c.GetString("user_id"))
		allowed, err := h.canAccessRoom(userID, roomID)
		if err != nil {
			response.FromError(c, err)
			return
		}
		if !allowed {
			response.Forbidden(c, "chat room access denied")
			return
		}
	}

	clients := h.hub.GetRoomClients(roomID)
	response.Success(c, gin.H{
		"room_id":      roomID,
		"client_count": len(clients),
		"clients":      clients,
	})
}

// GetStats returns websocket-wide stats.
// @Summary      Get websocket stats
// @Description  Returns total room and client counts
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

// SetupWebSocketRoutes registers websocket routes.
func SetupWebSocketRoutes(r *gin.Engine, hub *Hub, cfg *config.Config) {
	var chatRepo ChatRepository
	if db := database.GetDB(); db != nil {
		chatRepo = NewChatRepository(db)
	}

	handler := NewHandler(hub, cfg, chatRepo)
	if err := handler.ensureChatInitialized(); err != nil {
		logger.Warn("failed to initialize chat schema: %v", err)
	}

	ws := r.Group("/ws")
	ws.Use(middleware.AuthMiddleware(cfg))
	{
		ws.GET("/chat", handler.HandleChat)
		ws.GET(
			"/admin-chat",
			middleware.RequireUserTypes("TA", "A", "M", "G", "AG"),
			middleware.RequirePermission("admin.page.admin_chat.read"),
			handler.HandleAdminChat,
		)
		ws.GET(
			"/admin-chat-notify",
			middleware.RequireUserTypes("TA", "A", "M", "G", "AG"),
			middleware.RequirePermission("admin.page.admin_chat.read"),
			handler.HandleAdminChatNotify,
		)
	}

	api := r.Group("/api/ws")
	api.Use(middleware.AuthMiddleware(cfg))
	{
		api.GET("/room/:room_id", handler.GetRoomInfo)
		api.GET("/stats", handler.GetStats)
	}

	adminChat := r.Group("/api/admin/chat")
	adminChat.Use(middleware.AuthMiddleware(cfg))
	adminChat.Use(middleware.RequireUserTypes("TA", "A", "M", "G", "AG"))
	{
		adminChat.GET("/rooms", middleware.RequirePermission("admin.page.admin_chat.read"), handler.ListChatRooms)
		adminChat.GET("/users", middleware.RequirePermission("admin.page.admin_chat.read"), handler.ListChatUsers)
		adminChat.POST("/rooms/group", middleware.RequirePermission("admin.page.admin_chat.create"), handler.CreateGroupRoom)
		adminChat.POST("/rooms/direct/:target_id", middleware.RequirePermission("admin.page.admin_chat.create"), handler.CreateDirectRoom)
		adminChat.GET("/rooms/:room_key/messages", middleware.RequirePermission("admin.page.admin_chat.read"), handler.ListChatMessages)
		adminChat.POST("/rooms/:room_key/messages", middleware.RequirePermission("admin.page.admin_chat.create"), handler.CreateChatMessage)
		adminChat.POST("/rooms/:room_key/read", middleware.RequirePermission("admin.page.admin_chat.read"), handler.MarkChatRoomRead)
	}
}

func isManagedAdminRoom(roomID string) bool {
	return roomID == AdminLoungeRoomKey || strings.HasPrefix(roomID, "grp_") || strings.HasPrefix(roomID, "dm_")
}

func isValidChatRoomKey(roomKey string) bool {
	return chatRoomKeyPattern.MatchString(strings.TrimSpace(roomKey))
}

func buildNotifyRoomKey(userID string) string {
	return "notify:" + strings.TrimSpace(userID)
}

func (h *Handler) publishRoomUnreadSnapshots(roomKey string) {
	if h.chat == nil {
		return
	}

	roomKey = strings.TrimSpace(roomKey)
	if roomKey == "" {
		return
	}
	if err := h.ensureChatInitialized(); err != nil {
		logger.Warn("failed to initialize chat before unread publish: %v", err)
		return
	}
	snapshots, err := h.chat.ListRoomUnreadSnapshots(roomKey)
	if err != nil {
		logger.Warn("failed to publish room unread snapshots (room=%s): %v", roomKey, err)
		return
	}

	for _, snapshot := range snapshots {
		h.hub.Publish(&Message{
			Type:   "chat_unread",
			Room:   buildNotifyRoomKey(snapshot.UserID),
			UserID: snapshot.UserID,
			Content: gin.H{
				"room_key":     snapshot.RoomKey,
				"unread_count": snapshot.UnreadCount,
			},
		})
	}
}

func (h *Handler) ensureChatInitialized() error {
	if h.chat == nil {
		return appErrors.New("DATABASE_ERROR", "chat repository is not available")
	}

	h.initOnce.Do(func() {
		if err := h.chat.EnsureSchema(); err != nil {
			h.initErr = err
			return
		}
		if err := h.chat.EnsureDefaultAdminRoom(); err != nil {
			logger.Warn("chat default room seed skipped: %v", err)
		}
	})
	return h.initErr
}

func (h *Handler) canAccessRoom(userID, roomKey string) (bool, error) {
	userID = strings.TrimSpace(userID)
	roomKey = strings.TrimSpace(roomKey)
	if userID == "" {
		return false, appErrors.New("UNAUTHORIZED", "authentication context missing")
	}
	if roomKey == "" {
		return false, appErrors.New("BAD_REQUEST", "room_key is required")
	}
	if err := h.ensureChatInitialized(); err != nil {
		return false, err
	}

	if roomKey == AdminLoungeRoomKey {
		if err := h.chat.EnsureRoomMember(AdminLoungeRoomKey, userID); err != nil {
			return false, err
		}
	}

	return h.chat.IsRoomMember(roomKey, userID)
}
