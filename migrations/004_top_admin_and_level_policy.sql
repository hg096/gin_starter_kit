-- 004_top_admin_and_level_policy.sql
-- Introduces TA(top-admin) role, replaces AG with G, and adds global level policy setting.

CREATE TABLE IF NOT EXISTS `_a_system_settings` (
    `setting_key` VARCHAR(80) NOT NULL,
    `setting_value` VARCHAR(255) NOT NULL,
    `updated_by` VARCHAR(50) NULL,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`setting_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO `_a_permissions` (`permission_code`, `description`) VALUES
    ('admin.system.level_policy.manage', 'Manage global auth level policy')
ON DUPLICATE KEY UPDATE
    `description` = VALUES(`description`);

-- Normalize legacy visitor role code in user records.
UPDATE `_user`
SET `u_auth_type` = 'G'
WHERE `u_auth_type` = 'AG';

-- Promote legacy A+10 accounts to dedicated TA role.
UPDATE `_user`
SET `u_auth_type` = 'TA', `u_auth_level` = 0
WHERE `u_auth_type` = 'A' AND `u_auth_level` = 10;

-- If TA is still missing, promote one highest-priority admin role account.
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
WHERE ta.ta_count = 0;

-- Top-admin gets all permissions.
INSERT INTO `_a_user_permissions` (`u_id`, `permission_code`)
SELECT u.`u_id`, p.`permission_code`
FROM `_user` u
JOIN `_a_permissions` p
WHERE u.`u_auth_type` = 'TA'
ON DUPLICATE KEY UPDATE `permission_code` = VALUES(`permission_code`);

-- Non-top admin role baseline permissions.
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
ON DUPLICATE KEY UPDATE `permission_code` = VALUES(`permission_code`);

INSERT INTO `_a_system_settings` (`setting_key`, `setting_value`, `updated_by`) VALUES
    ('level_policy_enabled', '1', NULL)
ON DUPLICATE KEY UPDATE
    `setting_value` = VALUES(`setting_value`);
