package websocket

import "time"

const (
	// AdminLoungeRoomKey is the default shared room for admin accounts.
	AdminLoungeRoomKey = "admin-lounge"

	chatRoomTypeGroup  = "group"
	chatRoomTypeDirect = "direct"
)

// ChatRoom represents one chat room visible in admin chat UI.
type ChatRoom struct {
	RoomKey     string    `json:"room_key"`
	RoomType    string    `json:"room_type"`
	Name        string    `json:"name"`
	CreatedBy   string    `json:"created_by"`
	MemberCount int       `json:"member_count"`
	OnlineCount int       `json:"online_count"`
	UnreadCount int       `json:"unread_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ChatUser is a selectable admin user in chat screens.
type ChatUser struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AuthType  string `json:"auth_type"`
	AuthLevel int    `json:"auth_level"`
}

// ChatMessage is one persisted chat message (reply/comment supported by ParentID).
type ChatMessage struct {
	ID         int64     `json:"id"`
	RoomKey    string    `json:"room_key"`
	ParentID   *int64    `json:"parent_id,omitempty"`
	SenderID   string    `json:"sender_id"`
	SenderName string    `json:"sender_name"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

// ChatCreateGroupRoomRequest creates a group room.
type ChatCreateGroupRoomRequest struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids"`
}

// ChatCreateMessageRequest creates a room message.
type ChatCreateMessageRequest struct {
	Content  string `json:"content"`
	ParentID *int64 `json:"parent_id"`
}

// ChatMarkReadRequest marks a room as read up to last_message_id.
type ChatMarkReadRequest struct {
	LastMessageID int64 `json:"last_message_id"`
}

// ChatUnreadSnapshot is unread count per room for one user.
type ChatUnreadSnapshot struct {
	UserID      string `json:"user_id"`
	RoomKey     string `json:"room_key"`
	UnreadCount int    `json:"unread_count"`
}
