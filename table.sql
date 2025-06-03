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

CREATE TABLE `_menu_groups` (
	`mg_idx` INT NOT NULL AUTO_INCREMENT,
	`mg_label` VARCHAR(100) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`mg_order` INT NOT NULL DEFAULT '0',
	PRIMARY KEY (`mg_idx`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;

CREATE TABLE `_menu_items` (
	`mi_idx` INT NOT NULL AUTO_INCREMENT,
	`mi_group_id` INT NULL DEFAULT NULL,
	`mi_label` VARCHAR(100) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`mi_href` VARCHAR(255) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`mi_roles` JSON NULL DEFAULT NULL,
	`mi_order` INT NOT NULL DEFAULT '0',
	PRIMARY KEY (`mi_idx`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;


INSERT INTO `_menu_groups` (`mg_idx`, `mg_label`, `mg_order`) VALUES
	(1, '기본 메뉴', 1),
	(2, '게시물 관리', 2),
	(3, '광고 관리', 3),
	(4, '설정', 4);

INSERT INTO `_menu_items` (`mi_idx`, `mi_group_id`, `mi_label`, `mi_href`, `mi_roles`, `mi_order`) VALUES
	(1, 1, '대시보드', '/adm/dashboard', '["A", "M", "AG"]', 1),
	(2, 2, '공지사항', '/adm/posts/notice', '["A", "M", "AG"]', 1),
	(3, 2, '자주 묻는 질문', '/adm/posts/faq', '["A", "M", "AG"]', 2),
	(4, 3, '배너 설정', '/adm/ads/banner', '["A", "M"]', 1),
	(5, 3, '광고 승인', '/adm/ads/approval', '["A", "M"]', 2),
	(6, 4, '설정', '/adm/settings', '["A"]', 1),
	(7, 0, '로그아웃', '/adm/manage/logout', '["A", "M", "AG"]', 6);

