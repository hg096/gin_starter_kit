-- 005_admin_pages_dynamic.sql
-- Adds dynamic admin page registry and page-scoped CRUD permission seeds.

CREATE TABLE IF NOT EXISTS `_a_admin_pages` (
    `page_key` VARCHAR(48) NOT NULL,
    `title` VARCHAR(120) NOT NULL,
    `path` VARCHAR(255) NOT NULL,
    `description` VARCHAR(255) NOT NULL DEFAULT '',
    `group_key` VARCHAR(48) NOT NULL DEFAULT 'general',
    `group_label` VARCHAR(100) NOT NULL DEFAULT '일반',
    `group_order` INT NOT NULL DEFAULT 100,
    `visible_roles` JSON NULL,
    `icon` VARCHAR(40) NOT NULL DEFAULT '',
    `sort_order` INT NOT NULL DEFAULT 100,
    `is_enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `is_builtin` TINYINT(1) NOT NULL DEFAULT 0,
    `created_by` VARCHAR(50) NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`page_key`),
    UNIQUE KEY `uq_a_admin_pages_path` (`path`),
    KEY `idx_a_admin_pages_enabled_sort` (`is_enabled`, `sort_order`),
    KEY `idx_a_admin_pages_enabled_group_sort` (`is_enabled`, `group_order`, `sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO `_a_admin_pages` (`page_key`, `title`, `path`, `description`, `group_key`, `group_label`, `group_order`, `visible_roles`, `icon`, `sort_order`, `is_enabled`, `is_builtin`, `created_by`) VALUES
    ('dashboard', '대시보드', '/admin/dashboard', '운영 지표 대시보드', 'core', '운영', 10, JSON_ARRAY('TA','A','M','G'), 'D', 10, 1, 1, NULL),
    ('users', '사용자 관리', '/admin/users', '사용자/권한 관리', 'core', '운영', 10, JSON_ARRAY('TA','A'), 'U', 20, 1, 1, NULL)
ON DUPLICATE KEY UPDATE
    `title` = VALUES(`title`),
    `description` = VALUES(`description`),
    `group_key` = VALUES(`group_key`),
    `group_label` = VALUES(`group_label`),
    `group_order` = VALUES(`group_order`),
    `visible_roles` = VALUES(`visible_roles`),
    `icon` = VALUES(`icon`),
    `sort_order` = VALUES(`sort_order`),
    `is_enabled` = VALUES(`is_enabled`),
    `is_builtin` = VALUES(`is_builtin`);

INSERT INTO `_a_permissions` (`permission_code`, `description`) VALUES
    ('admin.page.manage', 'Manage admin page catalog'),
    ('admin.page.dashboard.read', 'Read page: 대시보드'),
    ('admin.page.dashboard.create', 'Create resources on page: 대시보드'),
    ('admin.page.dashboard.update', 'Update resources on page: 대시보드'),
    ('admin.page.dashboard.delete', 'Delete resources on page: 대시보드'),
    ('admin.page.users.read', 'Read page: 사용자 관리'),
    ('admin.page.users.create', 'Create resources on page: 사용자 관리'),
    ('admin.page.users.update', 'Update resources on page: 사용자 관리'),
    ('admin.page.users.delete', 'Delete resources on page: 사용자 관리')
ON DUPLICATE KEY UPDATE
    `description` = VALUES(`description`);

-- Keep backward compatibility: old read permissions map to built-in page read permissions.
INSERT INTO `_a_user_permissions` (`u_id`, `permission_code`)
SELECT up.`u_id`, 'admin.page.dashboard.read'
FROM `_a_user_permissions` up
WHERE up.`permission_code` = 'admin.stats.read'
ON DUPLICATE KEY UPDATE `permission_code` = VALUES(`permission_code`);

INSERT INTO `_a_user_permissions` (`u_id`, `permission_code`)
SELECT up.`u_id`, 'admin.page.users.read'
FROM `_a_user_permissions` up
WHERE up.`permission_code` = 'admin.account.read'
ON DUPLICATE KEY UPDATE `permission_code` = VALUES(`permission_code`);

-- Top-admins can always manage page catalog.
INSERT INTO `_a_user_permissions` (`u_id`, `permission_code`)
SELECT u.`u_id`, 'admin.page.manage'
FROM `_user` u
WHERE u.`u_auth_type` = 'TA'
ON DUPLICATE KEY UPDATE `permission_code` = VALUES(`permission_code`);
