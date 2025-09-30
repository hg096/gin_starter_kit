-- 블로그 테이블 생성
CREATE TABLE IF NOT EXISTS _blog (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(200) NOT NULL COMMENT '제목',
    content TEXT NOT NULL COMMENT '내용',
    author_id VARCHAR(50) NOT NULL COMMENT '작성자 ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '생성일시',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '수정일시',
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (author_id) REFERENCES _user(u_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='블로그 테이블';