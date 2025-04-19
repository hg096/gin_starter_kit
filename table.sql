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
	`u_idx` INT NOT NULL AUTO_INCREMENT,
	`u_id` VARCHAR(50) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`u_pass` TEXT NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`u_auth_type` VARCHAR(10) NULL DEFAULT 'U' COLLATE 'utf8mb4_general_ci',
	`u_auth_level` INT NULL DEFAULT '0',
	`u_email` VARCHAR(100) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`u_name` VARCHAR(50) NULL DEFAULT NULL COLLATE 'utf8mb4_general_ci',
	`u_regi_date` DATETIME NULL DEFAULT (now()),
	PRIMARY KEY (`u_idx`) USING BTREE,
	UNIQUE INDEX `u_id` (`u_id`) USING BTREE
)
COLLATE='utf8mb4_general_ci'
ENGINE=InnoDB
;

