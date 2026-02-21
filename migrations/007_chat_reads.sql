-- 007_chat_reads.sql
-- Adds per-user room read cursor for unread counts.

CREATE TABLE IF NOT EXISTS _chat_room_reads (
    crr_room_key VARCHAR(80) NOT NULL,
    crr_user_id VARCHAR(50) NOT NULL,
    crr_last_message_idx BIGINT NOT NULL DEFAULT 0,
    crr_updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (crr_room_key, crr_user_id),
    KEY idx_chat_room_reads_user (crr_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
