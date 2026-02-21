-- 006_chat_features.sql
-- Adds admin chat room/member/message tables for group + direct + reply(comment) chat.

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

CREATE TABLE IF NOT EXISTS _chat_room_members (
    crm_room_key VARCHAR(80) NOT NULL,
    crm_user_id VARCHAR(50) NOT NULL,
    crm_joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (crm_room_key, crm_user_id),
    KEY idx_chat_room_members_user (crm_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

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

INSERT INTO _chat_rooms (cr_key, cr_type, cr_name, cr_created_by)
VALUES ('admin-lounge', 'group', '관리자 라운지', 'system')
ON DUPLICATE KEY UPDATE
    cr_name = VALUES(cr_name),
    cr_updated_at = CURRENT_TIMESTAMP;

INSERT INTO _chat_room_members (crm_room_key, crm_user_id)
SELECT 'admin-lounge', u.u_id
FROM _user u
WHERE u.u_auth_type IN ('TA', 'A', 'M', 'G', 'AG')
ON DUPLICATE KEY UPDATE
    crm_joined_at = crm_joined_at;
