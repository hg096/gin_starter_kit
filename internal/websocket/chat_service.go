package websocket

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	appErrors "gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"sort"
	"strings"
	"sync"
	"time"
)

type ChatService interface {
	EnsureInitialized() error
	ListRooms(userID string) ([]ChatRoom, error)
	ListAdminUsers() ([]ChatUser, error)
	CreateGroupRoom(actorID string, req *ChatCreateGroupRoomRequest) (*ChatRoom, error)
	GetOrCreateDirectRoom(actorID, targetID string) (*ChatRoom, error)
	ListMessages(userID, roomKey string, limit int, beforeID int64) ([]ChatMessage, error)
	CreateMessage(userID, roomKey string, req *ChatCreateMessageRequest) (*ChatMessage, error)
	MarkRoomRead(userID, roomKey string, lastMessageID int64) error
	ListRoomUnreadSnapshots(roomKey string) ([]ChatUnreadSnapshot, error)
	CanAccessRoom(userID, roomKey string) (bool, error)
}

type chatService struct {
	repo     ChatRepository
	initOnce sync.Once
	initErr  error
}

func NewChatService(repo ChatRepository) ChatService {
	return &chatService{
		repo: repo,
	}
}

func (s *chatService) EnsureInitialized() error {
	if s.repo == nil {
		return appErrors.New("DATABASE_ERROR", "chat repository is not available")
	}

	s.initOnce.Do(func() {
		if err := s.repo.EnsureSchema(); err != nil {
			s.initErr = err
			return
		}
		if err := s.repo.EnsureDefaultAdminRoom(); err != nil {
			// Seed is best-effort. Do not block chat APIs on transient DB failures.
			logger.Warn("chat default room seed skipped: %v", err)
		}
	})
	return s.initErr
}

func (s *chatService) ListRooms(userID string) ([]ChatRoom, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, appErrors.New("UNAUTHORIZED", "authentication context missing")
	}
	if err := s.EnsureInitialized(); err != nil {
		return nil, err
	}

	if err := s.repo.EnsureRoomMember(AdminLoungeRoomKey, userID); err != nil {
		// Room membership sync is best-effort for read API stability.
		logger.Warn("skip admin-lounge membership sync for user=%s: %v", userID, err)
	}

	return s.repo.ListRoomsForUser(userID)
}

func (s *chatService) ListAdminUsers() ([]ChatUser, error) {
	if err := s.EnsureInitialized(); err != nil {
		return nil, err
	}
	return s.repo.ListAdminUsers()
}

func (s *chatService) CreateGroupRoom(actorID string, req *ChatCreateGroupRoomRequest) (*ChatRoom, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, appErrors.New("UNAUTHORIZED", "authentication context missing")
	}
	if req == nil {
		return nil, appErrors.New("BAD_REQUEST", "request body is required")
	}
	if err := s.EnsureInitialized(); err != nil {
		return nil, err
	}

	roomName := strings.TrimSpace(req.Name)
	if roomName == "" {
		return nil, appErrors.New("BAD_REQUEST", "group room name is required")
	}
	if len(roomName) > 120 {
		return nil, appErrors.New("BAD_REQUEST", "group room name must be <= 120 chars")
	}

	members := normalizeMemberIDs(actorID, req.MemberIDs)
	if len(members) < 2 {
		return nil, appErrors.New("BAD_REQUEST", "group room requires at least two members")
	}

	roomKey, err := buildGroupRoomKey()
	if err != nil {
		return nil, appErrors.Wrap(err, "INTERNAL_ERROR", "failed to generate room key")
	}

	return s.repo.CreateGroupRoom(actorID, roomKey, roomName, members)
}

func (s *chatService) GetOrCreateDirectRoom(actorID, targetID string) (*ChatRoom, error) {
	actorID = strings.TrimSpace(actorID)
	targetID = strings.TrimSpace(targetID)
	if actorID == "" {
		return nil, appErrors.New("UNAUTHORIZED", "authentication context missing")
	}
	if targetID == "" {
		return nil, appErrors.New("BAD_REQUEST", "target_id is required")
	}
	if strings.EqualFold(actorID, targetID) {
		return nil, appErrors.New("BAD_REQUEST", "cannot create direct room with yourself")
	}
	if err := s.EnsureInitialized(); err != nil {
		return nil, err
	}

	roomKey := buildDirectRoomKey(actorID, targetID)
	roomName := fmt.Sprintf("1:1 %s ↔ %s", actorID, targetID)
	return s.repo.GetOrCreateDirectRoom(actorID, roomKey, roomName, targetID)
}

func (s *chatService) ListMessages(userID, roomKey string, limit int, beforeID int64) ([]ChatMessage, error) {
	allowed, err := s.CanAccessRoom(userID, roomKey)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, appErrors.New("FORBIDDEN", "chat room access denied")
	}
	return s.repo.ListMessages(roomKey, limit, beforeID)
}

func (s *chatService) CreateMessage(userID, roomKey string, req *ChatCreateMessageRequest) (*ChatMessage, error) {
	if req == nil {
		return nil, appErrors.New("BAD_REQUEST", "request body is required")
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, appErrors.New("BAD_REQUEST", "message content is required")
	}
	if len(content) > 2000 {
		return nil, appErrors.New("BAD_REQUEST", "message content must be <= 2000 chars")
	}

	allowed, err := s.CanAccessRoom(userID, roomKey)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, appErrors.New("FORBIDDEN", "chat room access denied")
	}
	return s.repo.CreateMessage(roomKey, userID, content, req.ParentID)
}

func (s *chatService) MarkRoomRead(userID, roomKey string, lastMessageID int64) error {
	allowed, err := s.CanAccessRoom(userID, roomKey)
	if err != nil {
		return err
	}
	if !allowed {
		return appErrors.New("FORBIDDEN", "chat room access denied")
	}
	return s.repo.MarkRoomRead(roomKey, userID, lastMessageID)
}

func (s *chatService) ListRoomUnreadSnapshots(roomKey string) ([]ChatUnreadSnapshot, error) {
	if strings.TrimSpace(roomKey) == "" {
		return nil, appErrors.New("BAD_REQUEST", "room_key is required")
	}
	if err := s.EnsureInitialized(); err != nil {
		return nil, err
	}
	return s.repo.ListRoomUnreadSnapshots(roomKey)
}

func (s *chatService) CanAccessRoom(userID, roomKey string) (bool, error) {
	userID = strings.TrimSpace(userID)
	roomKey = strings.TrimSpace(roomKey)
	if userID == "" {
		return false, appErrors.New("UNAUTHORIZED", "authentication context missing")
	}
	if roomKey == "" {
		return false, appErrors.New("BAD_REQUEST", "room_key is required")
	}
	if err := s.EnsureInitialized(); err != nil {
		return false, err
	}

	if roomKey == AdminLoungeRoomKey {
		if err := s.repo.EnsureRoomMember(AdminLoungeRoomKey, userID); err != nil {
			return false, err
		}
	}

	return s.repo.IsRoomMember(roomKey, userID)
}

func normalizeMemberIDs(actorID string, memberIDs []string) []string {
	set := make(map[string]struct{}, len(memberIDs)+1)
	set[strings.TrimSpace(actorID)] = struct{}{}

	for _, memberID := range memberIDs {
		trimmed := strings.TrimSpace(memberID)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}

	members := make([]string, 0, len(set))
	for memberID := range set {
		if memberID == "" {
			continue
		}
		members = append(members, memberID)
	}
	sort.Strings(members)
	return members
}

func buildGroupRoomKey() (string, error) {
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	suffix := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("grp_%d_%s", time.Now().Unix(), suffix), nil
}

func buildDirectRoomKey(userA, userB string) string {
	a := normalizeRoomKeyPart(userA)
	b := normalizeRoomKeyPart(userB)
	if a > b {
		a, b = b, a
	}
	return "dm_" + a + "_" + b
}

func normalizeRoomKeyPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}

	builder := strings.Builder{}
	builder.Grow(len(value))
	for _, ch := range value {
		if ch >= 'a' && ch <= 'z' {
			builder.WriteRune(ch)
			continue
		}
		if ch >= '0' && ch <= '9' {
			builder.WriteRune(ch)
			continue
		}
		builder.WriteRune('_')
	}

	normalized := strings.Trim(builder.String(), "_")
	if normalized == "" {
		return "unknown"
	}
	return normalized
}
