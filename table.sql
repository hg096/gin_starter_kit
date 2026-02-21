
CREATE TABLE `_a_error_logs` (
	`el_where` TEXT NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`el_message` TEXT NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`el_sql` TEXT NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`el_regi_date` DATETIME NULL DEFAULT (now())
)
COMMENT='에러로그'
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;

CREATE TABLE `_user` (
	`u_idx` INT(10) NOT NULL AUTO_INCREMENT,
	`u_id` VARCHAR(50) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`u_pass` TEXT NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`u_auth_type` VARCHAR(10) NULL DEFAULT 'U' COLLATE 'utf8mb4_general_ci',
	`u_auth_level` INT(10) NULL DEFAULT '0',
	`u_status` VARCHAR(20) NOT NULL DEFAULT 'active' COLLATE 'utf8mb4_general_ci',
	`u_token_valid_after` DATETIME NULL DEFAULT (now()),
	`u_email` VARCHAR(100) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`u_name` VARCHAR(50) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`u_re_token` TEXT NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`u_memo` TEXT NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`u_regi_date` DATETIME NULL DEFAULT (now()),
	PRIMARY KEY (`u_idx`) USING BTREE,
	UNIQUE INDEX `u_id` (`u_id`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;

CREATE TABLE `_chat_messages` (
	`cm_idx` BIGINT NOT NULL AUTO_INCREMENT,
	`cm_room_id` VARCHAR(255) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`cm_sender_id` VARCHAR(50) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`cm_receiver_id` VARCHAR(50) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`cm_content` TEXT NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`cm_timestamp` TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP),
	PRIMARY KEY (`cm_idx`) USING BTREE,
	INDEX `cm_room_id` (`cm_room_id`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;

CREATE TABLE IF NOT EXISTS `_chat_rooms` (
	`cr_key` VARCHAR(80) NOT NULL,
	`cr_type` VARCHAR(20) NOT NULL COLLATE 'utf8mb4_general_ci',
	`cr_name` VARCHAR(120) NOT NULL COLLATE 'utf8mb4_general_ci',
	`cr_created_by` VARCHAR(50) NOT NULL COLLATE 'utf8mb4_general_ci',
	`cr_created_at` DATETIME NOT NULL DEFAULT (now()),
	`cr_updated_at` DATETIME NOT NULL DEFAULT (now()),
	PRIMARY KEY (`cr_key`) USING BTREE,
	INDEX `idx_chat_rooms_type` (`cr_type`) USING BTREE,
	INDEX `idx_chat_rooms_updated` (`cr_updated_at`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;

CREATE TABLE IF NOT EXISTS `_chat_room_members` (
	`crm_room_key` VARCHAR(80) NOT NULL COLLATE 'utf8mb4_general_ci',
	`crm_user_id` VARCHAR(50) NOT NULL COLLATE 'utf8mb4_general_ci',
	`crm_joined_at` DATETIME NOT NULL DEFAULT (now()),
	PRIMARY KEY (`crm_room_key`, `crm_user_id`) USING BTREE,
	INDEX `idx_chat_room_members_user` (`crm_user_id`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;

CREATE TABLE IF NOT EXISTS `_chat_room_messages` (
	`crm_idx` BIGINT NOT NULL AUTO_INCREMENT,
	`crm_room_key` VARCHAR(80) NOT NULL COLLATE 'utf8mb4_general_ci',
	`crm_parent_idx` BIGINT NULL DEFAULT NULL,
	`crm_sender_id` VARCHAR(50) NOT NULL COLLATE 'utf8mb4_general_ci',
	`crm_content` TEXT NOT NULL COLLATE 'utf8mb4_general_ci',
	`crm_created_at` DATETIME NOT NULL DEFAULT (now()),
	PRIMARY KEY (`crm_idx`) USING BTREE,
	INDEX `idx_chat_room_messages_room_idx` (`crm_room_key`, `crm_idx`) USING BTREE,
	INDEX `idx_chat_room_messages_parent` (`crm_parent_idx`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;

CREATE TABLE IF NOT EXISTS `_chat_room_reads` (
	`crr_room_key` VARCHAR(80) NOT NULL COLLATE 'utf8mb4_general_ci',
	`crr_user_id` VARCHAR(50) NOT NULL COLLATE 'utf8mb4_general_ci',
	`crr_last_message_idx` BIGINT NOT NULL DEFAULT '0',
	`crr_updated_at` DATETIME NOT NULL DEFAULT (now()),
	PRIMARY KEY (`crr_room_key`, `crr_user_id`) USING BTREE,
	INDEX `idx_chat_room_reads_user` (`crr_user_id`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;

CREATE TABLE IF NOT EXISTS `_a_permissions` (
  `permission_code` varchar(120) COLLATE utf8mb4_general_ci NOT NULL,
  `description` varchar(255) COLLATE utf8mb4_general_ci NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`permission_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DELETE FROM `_a_permissions`;
INSERT INTO `_a_permissions` (`permission_code`, `description`, `created_at`) VALUES
	('admin.account.delete', 'Delete user account', '2026-02-21 11:44:23'),
	('admin.account.level.manage', 'Manage auth type/level', '2026-02-21 11:44:23'),
	('admin.account.password.reset', 'Reset user password', '2026-02-21 11:44:23'),
	('admin.account.permission.manage', 'Manage assigned permissions', '2026-02-21 11:44:23'),
	('admin.account.profile.update', 'Update user profile fields', '2026-02-21 11:44:23'),
	('admin.account.read', 'Read user/admin accounts', '2026-02-21 11:44:23'),
	('admin.account.status.update', 'Update user account status', '2026-02-21 11:44:23'),
	('admin.allowlist.manage', 'Manage delegable permission allowlist', '2026-02-21 11:44:23'),
	('admin.audit.read', 'Read admin audit logs', '2026-02-21 20:32:38'),
	('admin.page.admin_chat.create', 'Create resources on page: 관리자 채팅', '2026-02-21 18:05:55'),
	('admin.page.admin_chat.delete', 'Delete resources on page: 관리자 채팅', '2026-02-21 18:05:55'),
	('admin.page.admin_chat.read', 'Read page: 관리자 채팅', '2026-02-21 18:05:55'),
	('admin.page.admin_chat.update', 'Update resources on page: 관리자 채팅', '2026-02-21 18:05:55'),
	('admin.page.blogs.create', 'Create resources on page: 블로그 관리', '2026-02-21 18:05:55'),
	('admin.page.blogs.delete', 'Delete resources on page: 블로그 관리', '2026-02-21 18:05:55'),
	('admin.page.blogs.read', 'Read page: 블로그 관리', '2026-02-21 18:05:55'),
	('admin.page.blogs.update', 'Update resources on page: 블로그 관리', '2026-02-21 18:05:55'),
	('admin.page.dashboard.create', 'Create resources on page: 대시보드', '2026-02-21 16:51:55'),
	('admin.page.dashboard.delete', 'Delete resources on page: 대시보드', '2026-02-21 16:51:55'),
	('admin.page.dashboard.read', 'Read page: 대시보드', '2026-02-21 16:51:55'),
	('admin.page.dashboard.update', 'Update resources on page: 대시보드', '2026-02-21 16:51:55'),
	('admin.page.manage', 'Manage admin page catalog', '2026-02-21 16:51:55'),
	('admin.page.users.create', 'Create resources on page: 사용자 관리', '2026-02-21 16:51:55'),
	('admin.page.users.delete', 'Delete resources on page: 사용자 관리', '2026-02-21 16:51:55'),
	('admin.page.users.read', 'Read page: 사용자 관리', '2026-02-21 16:51:55'),
	('admin.page.users.update', 'Update resources on page: 사용자 관리', '2026-02-21 16:51:55'),
	('admin.stats.read', 'Read admin dashboard stats', '2026-02-21 11:44:23'),
	('admin.system.level_policy.manage', 'Manage global auth level policy', '2026-02-21 16:18:03');


CREATE TABLE IF NOT EXISTS `_a_user_permissions` (
	`u_id` VARCHAR(50) NOT NULL COLLATE 'utf8mb4_general_ci',
	`permission_code` VARCHAR(120) NOT NULL COLLATE 'utf8mb4_general_ci',
	`created_at` DATETIME NOT NULL DEFAULT (now()),
	PRIMARY KEY (`u_id`, `permission_code`) USING BTREE,
	INDEX `idx_a_user_permissions_code` (`permission_code`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;

CREATE TABLE IF NOT EXISTS `_a_delegable_permissions` (
	`permission_code` VARCHAR(120) NOT NULL COLLATE 'utf8mb4_general_ci',
	`created_at` DATETIME NOT NULL DEFAULT (now()),
	PRIMARY KEY (`permission_code`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;

CREATE TABLE IF NOT EXISTS `_a_admin_audit_logs` (
	`aal_idx` BIGINT NOT NULL AUTO_INCREMENT,
	`actor_id` VARCHAR(50) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`target_user_id` VARCHAR(50) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`action` VARCHAR(120) NOT NULL COLLATE 'utf8mb4_general_ci',
	`status` VARCHAR(20) NOT NULL COLLATE 'utf8mb4_general_ci',
	`message` TEXT NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`ip_addr` VARCHAR(64) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`before_data` JSON NULL DEFAULT NULL,
	`after_data` JSON NULL DEFAULT NULL,
	`created_at` DATETIME NOT NULL DEFAULT (now()),
	PRIMARY KEY (`aal_idx`) USING BTREE,
	INDEX `idx_aal_actor` (`actor_id`) USING BTREE,
	INDEX `idx_aal_target` (`target_user_id`) USING BTREE,
	INDEX `idx_aal_action` (`action`) USING BTREE,
	INDEX `idx_aal_created` (`created_at`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;

CREATE TABLE IF NOT EXISTS `_a_system_settings` (
	`setting_key` VARCHAR(80) NOT NULL,
	`setting_value` VARCHAR(255) NOT NULL,
	`updated_by` VARCHAR(50) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`updated_at` DATETIME NOT NULL DEFAULT (now()),
	PRIMARY KEY (`setting_key`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;


CREATE TABLE IF NOT EXISTS _a_admin_pages (
  page_key VARCHAR(48) NOT NULL,
  title VARCHAR(120) NOT NULL,
  path VARCHAR(255) NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  group_key VARCHAR(48) NOT NULL DEFAULT 'general',
  group_label VARCHAR(100) NOT NULL DEFAULT '일반',
  group_order INT NOT NULL DEFAULT 100,
  visible_roles JSON NULL,
  icon VARCHAR(40) NOT NULL DEFAULT '',
  sort_order INT NOT NULL DEFAULT 100,
  is_enabled TINYINT(1) NOT NULL DEFAULT 1,
  is_builtin TINYINT(1) NOT NULL DEFAULT 0,
  created_by VARCHAR(50) NULL,
  created_at DATETIME NOT NULL DEFAULT (now()),
  updated_at DATETIME NOT NULL DEFAULT (now()),
  PRIMARY KEY (page_key),
  UNIQUE KEY uq_a_admin_pages_path (path),
  KEY idx_a_admin_pages_enabled_sort (is_enabled, sort_order),
  KEY idx_a_admin_pages_enabled_group_sort (is_enabled, group_order, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DELETE FROM _a_admin_pages;
INSERT INTO _a_admin_pages (page_key, title, path, description, group_key, group_label, group_order, visible_roles, icon, sort_order, is_enabled, is_builtin, created_by) VALUES
  ('dashboard', '대시보드', '/admin/dashboard', '운영 지표 대시보드', 'core', '운영', 10, JSON_ARRAY('TA', 'A', 'M', 'G'), 'D', 10, 1, 1, NULL),
  ('users', '사용자 관리', '/admin/users', '사용자/권한 관리', 'core', '운영', 10, JSON_ARRAY('TA', 'A'), 'U', 20, 1, 1, NULL),
  ('blogs', '블로그 관리', '/admin/blogs', '게시글 생성/수정/삭제 및 작성자 관리', 'content', '콘텐츠', 20, JSON_ARRAY('TA', 'A', 'M'), 'B', 30, 1, 1, NULL),
  ('admin_chat', '관리자 채팅', '/admin/chat', '관리자 전용 실시간 채팅', 'communication', '커뮤니케이션', 30, JSON_ARRAY('TA', 'A', 'M', 'G'), 'C', 40, 1, 1, NULL)
ON DUPLICATE KEY UPDATE
  title = VALUES(title),
  description = VALUES(description),
  group_key = VALUES(group_key),
  group_label = VALUES(group_label),
  group_order = VALUES(group_order),
  visible_roles = VALUES(visible_roles),
  icon = VALUES(icon),
  sort_order = VALUES(sort_order),
  is_enabled = VALUES(is_enabled),
  is_builtin = VALUES(is_builtin);

INSERT INTO `_chat_room_members` (`crm_room_key`, `crm_user_id`)
SELECT 'admin-lounge', `u_id`
FROM `_user`
WHERE `u_auth_type` IN ('TA', 'A', 'M', 'G', 'AG')
ON DUPLICATE KEY UPDATE
	`crm_joined_at` = `crm_joined_at`
;

-- Normalize legacy guest role code.
UPDATE `_user`
SET `u_auth_type` = 'G'
WHERE `u_auth_type` = 'AG'
;

-- Promote legacy A+10 accounts to TA.
UPDATE `_user`
SET `u_auth_type` = 'TA', `u_auth_level` = 0
WHERE `u_auth_type` = 'A' AND `u_auth_level` = 10
;

-- If TA is missing, promote one highest-priority admin role account.
UPDATE `_user` u
JOIN (
	SELECT `u_id`
	FROM `_user`
	WHERE `u_auth_type` IN ('A', 'M', 'G')
	ORDER BY `u_auth_level` DESC, `u_regi_date` ASC, `u_id` ASC
	LIMIT 1
) seed ON seed.`u_id` = u.`u_id`
LEFT JOIN (
	SELECT COUNT(*) AS ta_count
	FROM `_user`
	WHERE `u_auth_type` = 'TA'
) ta ON 1 = 1
SET u.`u_auth_type` = 'TA',
	u.`u_auth_level` = 0
WHERE ta.ta_count = 0
;

-- Top-admin gets all permissions.
INSERT INTO `_a_user_permissions` (`u_id`, `permission_code`)
SELECT u.`u_id`, p.`permission_code`
FROM `_user` u
JOIN `_a_permissions` p
WHERE u.`u_auth_type` = 'TA'
ON DUPLICATE KEY UPDATE `permission_code` = VALUES(`permission_code`)
;

-- Non-top admins get read-only baseline.
INSERT INTO `_a_user_permissions` (`u_id`, `permission_code`)
SELECT u.`u_id`, b.`permission_code`
FROM `_user` u
JOIN (
	    SELECT 'admin.stats.read' AS permission_code
	    UNION ALL
	    SELECT 'admin.account.read' AS permission_code
	    UNION ALL
	    SELECT 'admin.audit.read' AS permission_code
		) b
WHERE u.`u_auth_type` IN ('A', 'M', 'G')
ON DUPLICATE KEY UPDATE `permission_code` = VALUES(`permission_code`)
;

-- Initial delegable allow-list for general admins.
INSERT INTO `_a_delegable_permissions` (`permission_code`) VALUES
	('admin.account.status.update'),
	('admin.account.password.reset'),
	('admin.account.permission.manage')
ON DUPLICATE KEY UPDATE `permission_code` = VALUES(`permission_code`)
;

INSERT INTO `_a_system_settings` (`setting_key`, `setting_value`, `updated_by`) VALUES
	('level_policy_enabled', '1', NULL)
ON DUPLICATE KEY UPDATE `setting_value` = VALUES(`setting_value`)
;



