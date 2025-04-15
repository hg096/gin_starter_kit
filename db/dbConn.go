package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"gin_starter/model/core" // core 패키지 import (모듈 경로에 맞게 수정)

	_ "github.com/go-sql-driver/mysql"
)

// Conn는 애플리케이션 전역에서 사용 가능한 데이터베이스 연결 객체입니다.
var Conn *sql.DB

// InitDB는 주어진 DSN을 사용해 데이터베이스 연결을 초기화합니다.
func InitDB() {

	DB_USER := os.Getenv("DB_USER")
	DB_PASS := os.Getenv("DB_PASS")
	DB_HOST := os.Getenv("DB_HOST")
	DB_PORT := os.Getenv("DB_PORT")
	DB_NAME := os.Getenv("DB_NAME")

	// DSN 예시: "username:password@tcp(127.0.0.1:3306)/dbname?charset=utf8&parseTime=True&loc=Local"
	// fmt.Sprintf("Hello, %s! You are %d years old, living in %s, and working as a %s.", name, age, city, occupation)
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

	// core 패키지 내 전역 DB 변수 설정 (이 부분을 여기서 처리)
	core.SetDB(Conn)
}
