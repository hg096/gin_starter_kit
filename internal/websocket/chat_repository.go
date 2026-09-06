package websocket

import (
	"database/sql"
	stderrors "errors"
	"fmt"
	"gin_starter/pkg/db/database"
	appErrors "gin_starter/pkg/errors"
	"gin_starter/pkg/logger"
	"sort"
	"strings"
	"time"
)

const (
	createChatRoomsTableSQL = `
CREATE TABLE IF NOT EXISTS _chat_rooms (
	cr_key VARCHAR(80) NOT NULL,
	cr_type VARCHAR(20) NOT NULL,
	cr_name VARCHAR(120) NOT NULL,
	cr_created_by VARCHAR(50) NOT NULL,
	cr_created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	cr_updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (cr_key),
	KEY idx_chat_rooms_type (cr_type),
	KEY idx_chat_rooms_updated (cr_updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
`

	createChatRoomMembersTableSQL = `
CREATE TABLE IF NOT EXISTS _chat_room_members (
	crm_room_key VARCHAR(80) NOT NULL,
	crm_user_id VARCHAR(50) NOT NULL,
	crm_joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (crm_room_key, crm_user_id),
	KEY idx_chat_room_members_user (crm_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
`

	createChatRoomMessagesTableSQL = `
CREATE TABLE IF NOT EXISTS _chat_room_messages (
	crm_idx BIGINT NOT NULL AUTO_INCREMENT,
	crm_room_key VARCHAR(80) NOT NULL,
	crm_parent_idx BIGINT NULL,
	crm_sender_id VARCHAR(50) NOT NULL,
	crm_content TEXT NOT NULL,
	crm_created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (crm_idx),
	KEY idx_chat_room_messages_room_idx (crm_room_key, crm_idx),
	KEY idx_chat_room_messages_parent (crm_parent_idx)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
`

	createChatRoomReadsTableSQL = `
CREATE TABLE IF NOT EXISTS _chat_room_reads (
	crr_room_key VARCHAR(80) NOT NULL,
	crr_user_id VARCHAR(50) NOT NULL,
	crr_last_message_idx BIGINT NOT NULL DEFAULT 0,
	crr_updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (crr_room_key, crr_user_id),
	KEY idx_chat_room_reads_user (crr_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
`
)

type ChatRepository interface {
	EnsureSchema() error
	EnsureDefaultAdminRoom() error
	EnsureRoomMember(roomKey, userID string) error
	ListRoomsForUser(userID string) ([]ChatRoom, error)
	ListAdminUsers() ([]ChatUser, error)
	GetRoomByKey(roomKey string) (*ChatRoom, error)
	IsRoomMember(roomKey, userID string) (bool, error)
	CreateGroupRoom(actorID, roomKey, roomName string, members []string) (*ChatRoom, error)
	GetOrCreateDirectRoom(actorID, roomKey, roomName, targetID string) (*ChatRoom, error)
	ListMessages(roomKey string, limit int, beforeID int64) ([]ChatMessage, error)
	CreateMessage(roomKey, senderID, content string, parentID *int64) (*ChatMessage, error)
	MarkRoomRead(roomKey, userID string, lastMessageID int64) error
	ListRoomUnreadSnapshots(roomKey string) ([]ChatUnreadSnapshot, error)
}

type chatRepository struct {
	db   *database.DB
	base *database.Repository
}

func NewChatRepository(db *database.DB) ChatRepository {
	if db == nil {
		return &chatRepository{}
	}
	return &chatRepository{
		db:   db,
		base: database.NewRepository(db),
	}
}

func (r *chatRepository) EnsureSchema() error {
	if r.db == nil || r.base == nil {
		return appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	queries := []string{
		createChatRoomsTableSQL,
		createChatRoomMembersTableSQL,
		createChatRoomMessagesTableSQL,
		createChatRoomReadsTableSQL,
	}
	for _, query := range queries {
		if _, err := r.base.ExecSchema(query); err != nil {
			logger.Error("failed to ensure chat schema: %v", err)
			return appErrors.Wrap(err, "DATABASE_ERROR", "failed to ensure chat schema")
		}
	}
	return nil
}

func (r *chatRepository) EnsureDefaultAdminRoom() error {
	if r.db == nil || r.base == nil {
		return appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	return r.runTxWithRetry(func(tx *sql.Tx) error {
		if err := r.upsertRoomTx(tx, AdminLoungeRoomKey, chatRoomTypeGroup, "관리자 라운지", "system"); err != nil {
			return err
		}

		rows, err := tx.Query(`
			SELECT u_id
			FROM _user
			WHERE u_auth_type IN ('TA', 'A', 'M', 'G', 'AG')
		`)
		if err != nil {
			return appErrors.Wrap(err, "DATABASE_ERROR", "failed to load admin users for chat room seed")
		}

		adminUserIDs := make([]string, 0, 16)
		for rows.Next() {
			var userID string
			if scanErr := rows.Scan(&userID); scanErr != nil {
				rows.Close()
				return appErrors.Wrap(scanErr, "DATABASE_ERROR", "failed to map admin user for chat room seed")
			}
			adminUserIDs = append(adminUserIDs, userID)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return appErrors.Wrap(err, "DATABASE_ERROR", "failed to iterate admin users for chat room seed")
		}
		if err := rows.Close(); err != nil {
			return appErrors.Wrap(err, "DATABASE_ERROR", "failed to close admin users cursor")
		}

		for _, userID := range adminUserIDs {
			if err := r.upsertRoomMemberTx(tx, AdminLoungeRoomKey, userID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *chatRepository) EnsureRoomMember(roomKey, userID string) error {
	roomKey = strings.TrimSpace(roomKey)
	userID = strings.TrimSpace(userID)
	if roomKey == "" || userID == "" {
		return appErrors.New("BAD_REQUEST", "room_key and user_id are required")
	}
	if r.db == nil || r.base == nil {
		return appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	if _, err := r.base.Exec(
		`INSERT INTO _chat_room_members (crm_joined_at, crm_room_key, crm_user_id)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE crm_joined_at = crm_joined_at`,
		time.Now().Round(0),
		roomKey,
		userID,
	); err != nil {
		return appErrors.Wrap(err, "DATABASE_ERROR", "failed to ensure chat room member")
	}
	return nil
}

func (r *chatRepository) ListRoomsForUser(userID string) ([]ChatRoom, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, appErrors.New("BAD_REQUEST", "user_id is required")
	}
	if r.base == nil {
		return nil, appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	rows, err := r.base.Query(`
		SELECT
			r.cr_key,
			r.cr_type,
			r.cr_name,
			r.cr_created_by,
			r.cr_created_at,
			r.cr_updated_at,
			(
				SELECT COUNT(*)
				FROM _chat_room_members m2
				WHERE m2.crm_room_key = r.cr_key
			) AS member_count,
			(
				SELECT COUNT(*)
				FROM _chat_room_messages msg
				LEFT JOIN _chat_room_reads rd
					ON rd.crr_room_key = r.cr_key
					AND rd.crr_user_id = ?
				WHERE msg.crm_room_key = r.cr_key
					AND msg.crm_sender_id <> ?
					AND msg.crm_idx > COALESCE(rd.crr_last_message_idx, 0)
			) AS unread_count
		FROM _chat_rooms r
		INNER JOIN _chat_room_members m
			ON m.crm_room_key = r.cr_key
		WHERE m.crm_user_id = ?
		ORDER BY r.cr_updated_at DESC, r.cr_created_at DESC, r.cr_key ASC
	`, userID, userID, userID)
	if err != nil {
		return nil, appErrors.Wrap(err, "DATABASE_ERROR", "failed to list chat rooms")
	}
	defer rows.Close()

	rooms := make([]ChatRoom, 0, 16)
	for rows.Next() {
		var room ChatRoom
		if scanErr := rows.Scan(
			&room.RoomKey,
			&room.RoomType,
			&room.Name,
			&room.CreatedBy,
			&room.CreatedAt,
			&room.UpdatedAt,
			&room.MemberCount,
			&room.UnreadCount,
		); scanErr != nil {
			return nil, appErrors.Wrap(scanErr, "DATABASE_ERROR", "failed to map chat room")
		}
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		return nil, appErrors.Wrap(err, "DATABASE_ERROR", "failed to iterate chat rooms")
	}

	return rooms, nil
}

func (r *chatRepository) ListAdminUsers() ([]ChatUser, error) {
	if r.base == nil {
		return nil, appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	rows, err := r.base.Query(`
		SELECT
			u_id,
			COALESCE(NULLIF(TRIM(u_name), ''), u_id) AS u_name,
			u_auth_type,
			u_auth_level
		FROM _user
		WHERE u_auth_type IN ('TA', 'A', 'M', 'G', 'AG')
		ORDER BY
			CASE u_auth_type
				WHEN 'TA' THEN 0
				WHEN 'A' THEN 1
				WHEN 'M' THEN 2
				WHEN 'G' THEN 3
				WHEN 'AG' THEN 3
				ELSE 4
			END,
			u_auth_level DESC,
			u_id ASC
	`)
	if err != nil {
		return nil, appErrors.Wrap(err, "DATABASE_ERROR", "failed to list chat users")
	}
	defer rows.Close()

	users := make([]ChatUser, 0, 32)
	for rows.Next() {
		var chatUser ChatUser
		if scanErr := rows.Scan(&chatUser.ID, &chatUser.Name, &chatUser.AuthType, &chatUser.AuthLevel); scanErr != nil {
			return nil, appErrors.Wrap(scanErr, "DATABASE_ERROR", "failed to map chat user")
		}
		users = append(users, chatUser)
	}
	if err := rows.Err(); err != nil {
		return nil, appErrors.Wrap(err, "DATABASE_ERROR", "failed to iterate chat users")
	}

	return users, nil
}

func (r *chatRepository) GetRoomByKey(roomKey string) (*ChatRoom, error) {
	roomKey = strings.TrimSpace(roomKey)
	if roomKey == "" {
		return nil, appErrors.New("BAD_REQUEST", "room_key is required")
	}
	if r.base == nil {
		return nil, appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	query := `
		SELECT
			r.cr_key,
			r.cr_type,
			r.cr_name,
			r.cr_created_by,
			r.cr_created_at,
			r.cr_updated_at,
			(
				SELECT COUNT(*)
				FROM _chat_room_members m2
				WHERE m2.crm_room_key = r.cr_key
			) AS member_count
		FROM _chat_rooms r
		WHERE r.cr_key = ?
		LIMIT 1
	`

	var room ChatRoom
	err := r.base.QueryRow(query, roomKey).Scan(
		&room.RoomKey,
		&room.RoomType,
		&room.Name,
		&room.CreatedBy,
		&room.CreatedAt,
		&room.UpdatedAt,
		&room.MemberCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.New("NOT_FOUND", "chat room not found")
		}
		return nil, appErrors.Wrap(err, "DATABASE_ERROR", "failed to load chat room")
	}
	return &room, nil
}

func (r *chatRepository) IsRoomMember(roomKey, userID string) (bool, error) {
	roomKey = strings.TrimSpace(roomKey)
	userID = strings.TrimSpace(userID)
	if roomKey == "" || userID == "" {
		return false, appErrors.New("BAD_REQUEST", "room_key and user_id are required")
	}
	if r.base == nil {
		return false, appErrors.New("DATABASE_ERROR", "database is not initialized")
	}
	exists, err := r.base.Exists("_chat_room_members", "crm_room_key = ? AND crm_user_id = ?", roomKey, userID)
	if err != nil {
		return false, appErrors.Wrap(err, "DATABASE_ERROR", "failed to check room membership")
	}
	return exists, nil
}

func (r *chatRepository) CreateGroupRoom(actorID, roomKey, roomName string, members []string) (*ChatRoom, error) {
	if r.db == nil || r.base == nil {
		return nil, appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	if err := r.runTxWithRetry(func(tx *sql.Tx) error {
		if err := r.upsertRoomTx(tx, roomKey, chatRoomTypeGroup, roomName, actorID); err != nil {
			return err
		}

		if err := r.ensureAdminUsersExistTx(tx, members); err != nil {
			return err
		}

		for _, memberID := range members {
			if err := r.upsertRoomMemberTx(tx, roomKey, memberID); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return r.GetRoomByKey(roomKey)
}

func (r *chatRepository) GetOrCreateDirectRoom(actorID, roomKey, roomName, targetID string) (*ChatRoom, error) {
	if r.db == nil || r.base == nil {
		return nil, appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	if err := r.runTxWithRetry(func(tx *sql.Tx) error {
		if err := r.ensureAdminUsersExistTx(tx, []string{actorID, targetID}); err != nil {
			return err
		}

		if err := r.upsertRoomTx(tx, roomKey, chatRoomTypeDirect, roomName, actorID); err != nil {
			return err
		}

		if err := r.upsertRoomMemberTx(tx, roomKey, actorID); err != nil {
			return err
		}
		if err := r.upsertRoomMemberTx(tx, roomKey, targetID); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return r.GetRoomByKey(roomKey)
}

func (r *chatRepository) ListMessages(roomKey string, limit int, beforeID int64) ([]ChatMessage, error) {
	roomKey = strings.TrimSpace(roomKey)
	if roomKey == "" {
		return nil, appErrors.New("BAD_REQUEST", "room_key is required")
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if r.base == nil {
		return nil, appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	query := `
		SELECT
			m.crm_idx,
			m.crm_room_key,
			m.crm_parent_idx,
			m.crm_sender_id,
			COALESCE(NULLIF(TRIM(u.u_name), ''), m.crm_sender_id) AS sender_name,
			m.crm_content,
			m.crm_created_at
		FROM _chat_room_messages m
		LEFT JOIN _user u ON u.u_id = m.crm_sender_id
		WHERE m.crm_room_key = ?
	`
	args := []interface{}{roomKey}

	if beforeID > 0 {
		query += " AND m.crm_idx < ?"
		args = append(args, beforeID)
	}

	query += " ORDER BY m.crm_idx DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.base.Query(query, args...)
	if err != nil {
		return nil, appErrors.Wrap(err, "DATABASE_ERROR", "failed to list chat messages")
	}
	defer rows.Close()

	messages := make([]ChatMessage, 0, limit)
	for rows.Next() {
		var message ChatMessage
		var parentID sql.NullInt64
		if scanErr := rows.Scan(
			&message.ID,
			&message.RoomKey,
			&parentID,
			&message.SenderID,
			&message.SenderName,
			&message.Content,
			&message.CreatedAt,
		); scanErr != nil {
			return nil, appErrors.Wrap(scanErr, "DATABASE_ERROR", "failed to map chat message")
		}
		if parentID.Valid {
			value := parentID.Int64
			message.ParentID = &value
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, appErrors.Wrap(err, "DATABASE_ERROR", "failed to iterate chat messages")
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func (r *chatRepository) CreateMessage(roomKey, senderID, content string, parentID *int64) (*ChatMessage, error) {
	roomKey = strings.TrimSpace(roomKey)
	senderID = strings.TrimSpace(senderID)
	content = strings.TrimSpace(content)
	if roomKey == "" || senderID == "" {
		return nil, appErrors.New("BAD_REQUEST", "room_key and sender_id are required")
	}
	if content == "" {
		return nil, appErrors.New("BAD_REQUEST", "message content is required")
	}
	if r.db == nil || r.base == nil {
		return nil, appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	var createdMessageID int64
	if err := r.runTxWithRetry(func(tx *sql.Tx) error {
		if parentID != nil {
			var existsCount int
			if scanErr := r.base.QueryRowTx(
				tx,
				"SELECT COUNT(*) FROM _chat_room_messages WHERE crm_idx = ? AND crm_room_key = ?",
				*parentID,
				roomKey,
			).Scan(&existsCount); scanErr != nil {
				return appErrors.Wrap(scanErr, "DATABASE_ERROR", "failed to validate parent message")
			}
			if existsCount == 0 {
				return appErrors.New("BAD_REQUEST", "parent message not found in this room")
			}
		}

		insertData := map[string]interface{}{
			"crm_room_key":  roomKey,
			"crm_sender_id": senderID,
			"crm_content":   content,
		}
		if parentID != nil {
			insertData["crm_parent_idx"] = *parentID
		}

		messageID, err := r.base.Tx(tx).Insert("_chat_room_messages", insertData)
		if err != nil {
			return appErrors.Wrap(err, "DATABASE_ERROR", "failed to create chat message")
		}
		createdMessageID = messageID

		affected, err := r.base.Tx(tx).Update("_chat_rooms", map[string]interface{}{
			"cr_updated_at": time.Now().Round(0),
		}, "cr_key = ?", roomKey)
		if err != nil {
			return appErrors.Wrap(err, "DATABASE_ERROR", "failed to update chat room timestamp")
		}
		if affected == 0 {
			return appErrors.New("NOT_FOUND", "chat room not found")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return r.getMessageByID(createdMessageID)
}

func (r *chatRepository) MarkRoomRead(roomKey, userID string, lastMessageID int64) error {
	roomKey = strings.TrimSpace(roomKey)
	userID = strings.TrimSpace(userID)
	if roomKey == "" || userID == "" {
		return appErrors.New("BAD_REQUEST", "room_key and user_id are required")
	}
	if r.base == nil {
		return appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	if lastMessageID <= 0 {
		if err := r.base.QueryRow(
			"SELECT COALESCE(MAX(crm_idx), 0) FROM _chat_room_messages WHERE crm_room_key = ?",
			roomKey,
		).Scan(&lastMessageID); err != nil {
			return appErrors.Wrap(err, "DATABASE_ERROR", "failed to resolve latest chat message")
		}
	}

	if _, err := r.base.Exec(
		`INSERT INTO _chat_room_reads (crr_room_key, crr_user_id, crr_last_message_idx, crr_updated_at)
		 VALUES (?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   crr_last_message_idx = GREATEST(crr_last_message_idx, VALUES(crr_last_message_idx)),
		   crr_updated_at = VALUES(crr_updated_at)`,
		roomKey,
		userID,
		lastMessageID,
		time.Now().Round(0),
	); err != nil {
		return appErrors.Wrap(err, "DATABASE_ERROR", "failed to mark chat room read")
	}

	return nil
}

func (r *chatRepository) ListRoomUnreadSnapshots(roomKey string) ([]ChatUnreadSnapshot, error) {
	roomKey = strings.TrimSpace(roomKey)
	if roomKey == "" {
		return nil, appErrors.New("BAD_REQUEST", "room_key is required")
	}
	if r.base == nil {
		return nil, appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	rows, err := r.base.Query(
		`SELECT
			m.crm_user_id,
			COALESCE(SUM(
				CASE
					WHEN msg.crm_idx IS NULL THEN 0
					WHEN msg.crm_sender_id = m.crm_user_id THEN 0
					WHEN msg.crm_idx > COALESCE(rd.crr_last_message_idx, 0) THEN 1
					ELSE 0
				END
			), 0) AS unread_count
		FROM _chat_room_members m
		LEFT JOIN _chat_room_reads rd
			ON rd.crr_room_key = m.crm_room_key
			AND rd.crr_user_id = m.crm_user_id
		LEFT JOIN _chat_room_messages msg
			ON msg.crm_room_key = m.crm_room_key
		WHERE m.crm_room_key = ?
		GROUP BY m.crm_user_id`,
		roomKey,
	)
	if err != nil {
		return nil, appErrors.Wrap(err, "DATABASE_ERROR", "failed to list room unread snapshots")
	}
	defer rows.Close()

	snapshots := make([]ChatUnreadSnapshot, 0, 8)
	for rows.Next() {
		var snapshot ChatUnreadSnapshot
		snapshot.RoomKey = roomKey
		if scanErr := rows.Scan(&snapshot.UserID, &snapshot.UnreadCount); scanErr != nil {
			return nil, appErrors.Wrap(scanErr, "DATABASE_ERROR", "failed to map room unread snapshot")
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, appErrors.Wrap(err, "DATABASE_ERROR", "failed to iterate room unread snapshots")
	}

	return snapshots, nil
}

func (r *chatRepository) getMessageByID(messageID int64) (*ChatMessage, error) {
	query := `
		SELECT
			m.crm_idx,
			m.crm_room_key,
			m.crm_parent_idx,
			m.crm_sender_id,
			COALESCE(NULLIF(TRIM(u.u_name), ''), m.crm_sender_id) AS sender_name,
			m.crm_content,
			m.crm_created_at
		FROM _chat_room_messages m
		LEFT JOIN _user u ON u.u_id = m.crm_sender_id
		WHERE m.crm_idx = ?
		LIMIT 1
	`

	var message ChatMessage
	var parentID sql.NullInt64
	err := r.base.QueryRow(query, messageID).Scan(
		&message.ID,
		&message.RoomKey,
		&parentID,
		&message.SenderID,
		&message.SenderName,
		&message.Content,
		&message.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.New("NOT_FOUND", "chat message not found")
		}
		return nil, appErrors.Wrap(err, "DATABASE_ERROR", "failed to load chat message")
	}
	if parentID.Valid {
		value := parentID.Int64
		message.ParentID = &value
	}
	return &message, nil
}

func (r *chatRepository) upsertRoomTx(tx *sql.Tx, roomKey, roomType, roomName, createdBy string) error {
	now := time.Now().Round(0)

	updateData := map[string]interface{}{
		"cr_type":       roomType,
		"cr_name":       roomName,
		"cr_created_by": createdBy,
		"cr_updated_at": now,
	}
	affected, err := r.base.Tx(tx).Update("_chat_rooms", updateData, "cr_key = ?", roomKey)
	if err != nil {
		return appErrors.Wrap(err, "DATABASE_ERROR", "failed to update chat room")
	}
	if affected > 0 {
		return nil
	}

	insertData := map[string]interface{}{
		"cr_key":        roomKey,
		"cr_type":       roomType,
		"cr_name":       roomName,
		"cr_created_by": createdBy,
		"cr_created_at": now,
		"cr_updated_at": now,
	}
	if _, err := r.base.Tx(tx).Insert("_chat_rooms", insertData); err != nil && !isDuplicateKeyError(err) {
		return appErrors.Wrap(err, "DATABASE_ERROR", "failed to create chat room")
	}
	return nil
}

func (r *chatRepository) upsertRoomMemberTx(tx *sql.Tx, roomKey, userID string) error {
	if _, err := r.base.Tx(tx).Exec(
		`INSERT INTO _chat_room_members (crm_joined_at, crm_room_key, crm_user_id)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE crm_joined_at = crm_joined_at`,
		time.Now().Round(0),
		roomKey,
		userID,
	); err != nil {
		return appErrors.Wrap(err, "DATABASE_ERROR", "failed to upsert chat room member")
	}
	return nil
}

func (r *chatRepository) ensureAdminUsersExistTx(tx *sql.Tx, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(userIDs))
	seen := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		trimmed := strings.TrimSpace(userID)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return appErrors.New("BAD_REQUEST", "member_ids is required")
	}

	sort.Strings(normalized)
	for _, userID := range normalized {
		var existsCount int
		err := r.base.QueryRowTx(
			tx,
			"SELECT COUNT(*) FROM _user WHERE u_id = ? AND u_auth_type IN ('TA', 'A', 'M', 'G', 'AG')",
			userID,
		).Scan(&existsCount)
		if err != nil {
			return appErrors.Wrap(err, "DATABASE_ERROR", "failed to validate chat user")
		}
		if existsCount == 0 {
			return appErrors.New("BAD_REQUEST", fmt.Sprintf("invalid admin user: %s", userID))
		}
	}
	return nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") || strings.Contains(msg, "error 1062")
}

func (r *chatRepository) runTxWithRetry(fn func(tx *sql.Tx) error) error {
	if r.db == nil {
		return appErrors.New("DATABASE_ERROR", "database is not initialized")
	}

	const maxAttempts = 2
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		tx, beginErr := r.db.BeginTx()
		if beginErr != nil {
			wrapped := appErrors.Wrap(beginErr, "DATABASE_ERROR", "failed to begin transaction")
			if isRetryableDBConnError(beginErr) && attempt < maxAttempts {
				logger.Warn("retrying chat transaction after begin failure: attempt=%d err=%v", attempt, beginErr)
				if pingErr := r.db.Ping(); pingErr != nil {
					logger.Warn("db ping failed during retry recovery (begin): %v", pingErr)
				}
				time.Sleep(50 * time.Millisecond)
				lastErr = wrapped
				continue
			}
			return wrapped
		}

		execErr := fn(tx)
		if execErr != nil {
			database.RollbackTx(tx)
			if isRetryableDBConnError(execErr) && attempt < maxAttempts {
				logger.Warn("retrying chat transaction after execution failure: attempt=%d err=%v", attempt, execErr)
				if pingErr := r.db.Ping(); pingErr != nil {
					logger.Warn("db ping failed during retry recovery (execution): %v", pingErr)
				}
				time.Sleep(50 * time.Millisecond)
				lastErr = execErr
				continue
			}
			return execErr
		}

		commitErr := database.CommitTx(tx)
		if commitErr != nil {
			wrapped := appErrors.Wrap(commitErr, "DATABASE_ERROR", "failed to commit transaction")
			if isRetryableDBConnError(commitErr) && attempt < maxAttempts {
				logger.Warn("retrying chat transaction after commit failure: attempt=%d err=%v", attempt, commitErr)
				if pingErr := r.db.Ping(); pingErr != nil {
					logger.Warn("db ping failed during retry recovery (commit): %v", pingErr)
				}
				time.Sleep(50 * time.Millisecond)
				lastErr = wrapped
				continue
			}
			return wrapped
		}

		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return appErrors.New("DATABASE_ERROR", "failed to execute chat transaction")
}

func isRetryableDBConnError(err error) bool {
	if err == nil {
		return false
	}

	var appErr *appErrors.AppError
	if stderrors.As(err, &appErr) && appErr.Err != nil {
		if isRetryableDBConnError(appErr.Err) {
			return true
		}
	}

	msg := strings.ToLower(err.Error())
	tokens := []string{
		"driver: bad connection",
		"bad connection",
		"invalid connection",
		"broken pipe",
		"connection reset",
		"server has gone away",
		"lost connection",
	}
	for _, token := range tokens {
		if strings.Contains(msg, token) {
			return true
		}
	}
	return false
}
