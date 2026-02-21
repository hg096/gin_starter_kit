-- 003_authz_permissions.sql
-- Adds permission model, token invalidation fields, and admin audit logging.

ALTER TABLE `_user`
    ADD COLUMN `u_status` VARCHAR(20) NOT NULL DEFAULT 'active' AFTER `u_auth_level`,
    ADD COLUMN `u_token_valid_after` DATETIME NULL DEFAULT CURRENT_TIMESTAMP AFTER `u_status`;

UPDATE `_user`
SET `u_status` = 'active'
WHERE `u_status` IS NULL OR `u_status` = '';

UPDATE `_user`
SET `u_token_valid_after` = NOW()
WHERE `u_token_valid_after` IS NULL;

CREATE TABLE IF NOT EXISTS `_a_permissions` (
    `permission_code` VARCHAR(120) NOT NULL,
    `description` VARCHAR(255) NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`permission_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `_a_user_permissions` (
    `u_id` VARCHAR(50) NOT NULL,
    `permission_code` VARCHAR(120) NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`u_id`, `permission_code`),
    KEY `idx_a_user_permissions_code` (`permission_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `_a_delegable_permissions` (
    `permission_code` VARCHAR(120) NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`permission_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `_a_admin_audit_logs` (
    `aal_idx` BIGINT NOT NULL AUTO_INCREMENT,
    `actor_id` VARCHAR(50) NULL,
    `target_user_id` VARCHAR(50) NULL,
    `action` VARCHAR(120) NOT NULL,
    `status` VARCHAR(20) NOT NULL,
    `message` TEXT NULL,
    `ip_addr` VARCHAR(64) NULL,
    `before_data` JSON NULL,
    `after_data` JSON NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`aal_idx`),
    KEY `idx_aal_actor` (`actor_id`),
    KEY `idx_aal_target` (`target_user_id`),
    KEY `idx_aal_action` (`action`),
    KEY `idx_aal_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO `_a_permissions` (`permission_code`, `description`) VALUES
    ('admin.stats.read', 'Read admin dashboard stats'),
    ('admin.account.read', 'Read user/admin accounts'),
    ('admin.audit.read', 'Read admin audit logs'),
    ('admin.account.profile.update', 'Update user profile fields'),
    ('admin.account.status.update', 'Update user account status'),
    ('admin.account.password.reset', 'Reset user password'),
    ('admin.account.permission.manage', 'Manage assigned permissions'),
    ('admin.account.level.manage', 'Manage auth type/level'),
    ('admin.account.delete', 'Delete user account'),
    ('admin.page.manage', 'Manage admin page catalog'),
    ('admin.allowlist.manage', 'Manage delegable permission allowlist')
ON DUPLICATE KEY UPDATE
    `description` = VALUES(`description`);

-- Super-admin bootstrap: promote highest-level admin to level 10.
UPDATE `_user` u
JOIN (
    SELECT `u_id`
    FROM `_user`
    WHERE `u_auth_type` = 'A'
    ORDER BY `u_auth_level` DESC, `u_regi_date` ASC, `u_id` ASC
    LIMIT 1
) seed ON seed.`u_id` = u.`u_id`
SET u.`u_auth_level` = 10;

-- Super-admin gets all permissions.
INSERT INTO `_a_user_permissions` (`u_id`, `permission_code`)
SELECT u.`u_id`, p.`permission_code`
FROM `_user` u
JOIN `_a_permissions` p
WHERE u.`u_auth_type` = 'A' AND u.`u_auth_level` = 10
ON DUPLICATE KEY UPDATE `permission_code` = VALUES(`permission_code`);

-- Non-super admins get read-only baseline.
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
WHERE u.`u_auth_type` = 'A' AND u.`u_auth_level` < 10
ON DUPLICATE KEY UPDATE `permission_code` = VALUES(`permission_code`);

-- Initial delegable allow-list for general admins.
INSERT INTO `_a_delegable_permissions` (`permission_code`) VALUES
    ('admin.account.status.update'),
    ('admin.account.password.reset'),
    ('admin.account.permission.manage')
ON DUPLICATE KEY UPDATE `permission_code` = VALUES(`permission_code`);
