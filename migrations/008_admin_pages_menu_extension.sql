-- 008_admin_pages_menu_extension.sql
-- Expands _a_admin_pages to absorb legacy _menu_groups/_menu_items semantics.

-- Add columns only when missing (compatible with MySQL versions without ADD COLUMN IF NOT EXISTS).
SET @col_exists := (
    SELECT COUNT(1)
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = '_a_admin_pages'
      AND column_name = 'group_key'
);
SET @col_sql := IF(
    @col_exists = 0,
    'ALTER TABLE `_a_admin_pages` ADD COLUMN `group_key` VARCHAR(48) NOT NULL DEFAULT ''general'' AFTER `description`',
    'SELECT 1'
);
PREPARE stmt_col FROM @col_sql;
EXECUTE stmt_col;
DEALLOCATE PREPARE stmt_col;

SET @col_exists := (
    SELECT COUNT(1)
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = '_a_admin_pages'
      AND column_name = 'group_label'
);
SET @col_sql := IF(
    @col_exists = 0,
    'ALTER TABLE `_a_admin_pages` ADD COLUMN `group_label` VARCHAR(100) NOT NULL DEFAULT ''일반'' AFTER `group_key`',
    'SELECT 1'
);
PREPARE stmt_col FROM @col_sql;
EXECUTE stmt_col;
DEALLOCATE PREPARE stmt_col;

SET @col_exists := (
    SELECT COUNT(1)
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = '_a_admin_pages'
      AND column_name = 'group_order'
);
SET @col_sql := IF(
    @col_exists = 0,
    'ALTER TABLE `_a_admin_pages` ADD COLUMN `group_order` INT NOT NULL DEFAULT 100 AFTER `group_label`',
    'SELECT 1'
);
PREPARE stmt_col FROM @col_sql;
EXECUTE stmt_col;
DEALLOCATE PREPARE stmt_col;

SET @col_exists := (
    SELECT COUNT(1)
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = '_a_admin_pages'
      AND column_name = 'visible_roles'
);
SET @col_sql := IF(
    @col_exists = 0,
    'ALTER TABLE `_a_admin_pages` ADD COLUMN `visible_roles` JSON NULL AFTER `group_order`',
    'SELECT 1'
);
PREPARE stmt_col FROM @col_sql;
EXECUTE stmt_col;
DEALLOCATE PREPARE stmt_col;

-- Keep dashboard canonical route path in sync with current routing.
UPDATE `_a_admin_pages`
SET `path` = '/admin/dashboard'
WHERE `page_key` = 'dashboard' AND `path` = '/admin';

-- Built-in grouping defaults.
UPDATE `_a_admin_pages`
SET
    `group_key` = 'core',
    `group_label` = '운영',
    `group_order` = 10,
    `visible_roles` = JSON_ARRAY('TA','A','M','G')
WHERE `page_key` = 'dashboard';

UPDATE `_a_admin_pages`
SET
    `group_key` = 'core',
    `group_label` = '운영',
    `group_order` = 10,
    `visible_roles` = JSON_ARRAY('TA','A')
WHERE `page_key` = 'users';

UPDATE `_a_admin_pages`
SET
    `group_key` = 'content',
    `group_label` = '콘텐츠',
    `group_order` = 20,
    `visible_roles` = JSON_ARRAY('TA','A','M')
WHERE `page_key` = 'blogs';

UPDATE `_a_admin_pages`
SET
    `group_key` = 'communication',
    `group_label` = '커뮤니케이션',
    `group_order` = 30,
    `visible_roles` = JSON_ARRAY('TA','A','M','G')
WHERE `page_key` = 'admin_chat';

-- Generic fallback for existing custom pages.
UPDATE `_a_admin_pages`
SET
    `group_key` = COALESCE(NULLIF(TRIM(`group_key`), ''), 'custom'),
    `group_label` = COALESCE(NULLIF(TRIM(`group_label`), ''), '기타'),
    `group_order` = CASE WHEN `group_order` < 1 THEN 90 ELSE `group_order` END
WHERE 1 = 1;

-- Add grouped-order index only when missing.
SET @idx_exists := (
    SELECT COUNT(1)
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = '_a_admin_pages'
      AND index_name = 'idx_a_admin_pages_enabled_group_sort'
);
SET @idx_sql := IF(
    @idx_exists = 0,
    'ALTER TABLE `_a_admin_pages` ADD INDEX `idx_a_admin_pages_enabled_group_sort` (`is_enabled`, `group_order`, `sort_order`)',
    'SELECT 1'
);
PREPARE stmt_idx FROM @idx_sql;
EXECUTE stmt_idx;
DEALLOCATE PREPARE stmt_idx;
