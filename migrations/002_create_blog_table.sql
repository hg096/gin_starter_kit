CREATE TABLE `_blog` (
	`id` BIGINT NOT NULL AUTO_INCREMENT,
	`title` VARCHAR(200) NULL DEFAULT NULL COMMENT '제목' COLLATE 'utf8mb4_unicode_ci',
	`content` TEXT NULL DEFAULT NULL COMMENT '내용' COLLATE 'utf8mb4_unicode_ci',
	`author_id` VARCHAR(50) NULL DEFAULT NULL COMMENT '작성자 ID' COLLATE 'utf8mb4_unicode_ci',
	`created_at` TIMESTAMP NULL DEFAULT (CURRENT_TIMESTAMP) COMMENT '생성일시',
	`updated_at` TIMESTAMP NULL DEFAULT (CURRENT_TIMESTAMP) ON UPDATE CURRENT_TIMESTAMP COMMENT '수정일시',
	PRIMARY KEY (`id`) USING BTREE,
	INDEX `idx_author_id` (`author_id`) USING BTREE,
	INDEX `idx_created_at` (`created_at`) USING BTREE
)
COMMENT='블로그 테이블'
COLLATE='utf8mb4_unicode_ci'
ENGINE=InnoDB
;
