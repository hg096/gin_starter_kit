package database

import (
	"context"
	"database/sql"
	"fmt"
	"gin_starter/internal/config"
	"gin_starter/pkg/logger"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB 데이터베이스 연결 래퍼
type DB struct {
	*sql.DB
}

var instance *DB

// Connect 데이터베이스 연결
func Connect(cfg *config.Config) (*DB, error) {
	dsn := cfg.GetDSN()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("데이터베이스 열기 실패: %w", err)
	}

	// 연결 풀 설정
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// 연결 확인
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("데이터베이스 연결 실패: %w", err)
	}

	instance = &DB{DB: db}
	logger.Info("✅ MySQL 연결 성공 (호스트: %s, 데이터베이스: %s)", cfg.Database.Host, cfg.Database.Database)

	return instance, nil
}

// GetDB 싱글톤 DB 인스턴스 반환
func GetDB() *DB {
	return instance
}

// Close 데이터베이스 연결 종료
func (db *DB) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}

// HealthCheck 데이터베이스 헬스 체크
func (db *DB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("데이터베이스 헬스 체크 실패: %w", err)
	}

	return nil
}

// BeginTx 트랜잭션 시작
func (db *DB) BeginTx() (*sql.Tx, error) {
	tx, err := db.Begin()
	if err != nil {
		logger.Error("트랜잭션 시작 실패: %v", err)
		return nil, err
	}
	return tx, nil
}

// CommitTx 트랜잭션 커밋
func CommitTx(tx *sql.Tx) error {
	if tx == nil {
		return fmt.Errorf("트랜잭션이 nil입니다")
	}

	if err := tx.Commit(); err != nil {
		logger.Error("트랜잭션 커밋 실패: %v", err)
		return err
	}
	return nil
}

// RollbackTx 트랜잭션 롤백
func RollbackTx(tx *sql.Tx) {
	if tx == nil {
		return
	}

	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
		logger.Error("트랜잭션 롤백 실패: %v", err)
	}
}