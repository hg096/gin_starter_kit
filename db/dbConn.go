package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"gin_starter/model/core"

	_ "github.com/go-sql-driver/mysql"
)

// Conn는 애플리케이션 전역에서 사용 가능한 데이터베이스 연결 객체
var Conn *sql.DB

// 데이터베이스 연결을 초기화
func InitDB() {

	DB_USER := os.Getenv("DB_USER")
	DB_PASS := os.Getenv("DB_PASS")
	DB_HOST := os.Getenv("DB_HOST")
	DB_PORT := os.Getenv("DB_PORT")
	DB_NAME := os.Getenv("DB_NAME")

	// DSN 예시: "username:password@tcp(127.0.0.1:3306)/dbname?charset=utf8&parseTime=True&loc=Local"
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", DB_USER, DB_PASS, DB_HOST, DB_PORT, DB_NAME)

	var err error
	Conn, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("[종료] 데이터베이스 열기 에러: %v", err)
	}
	if err = Conn.Ping(); err != nil {
		log.Fatalf("[종료] 데이터베이스 연결 에러: %v", err)
	}
	log.Println("MySQL 연결 성공")

	// core 패키지 내 전역 DB 변수 설정
	core.SetDB(Conn)
}
