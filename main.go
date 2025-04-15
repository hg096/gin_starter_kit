package main

// env 설치
// go get github.com/joho/godotenv

import (
	"log"
	"os"

	"gin_starter/db"     // 실제 모듈 경로로 수정하세요.
	"gin_starter/routes" // 실제 모듈 경로로 수정하세요.

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	// .env 파일 로드
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// .env 파일에 정의된 PORT 값을 가져옵니다.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // env에 PORT가 없으면 기본 포트 8080 사용
	}

	db.InitDB() // db 폴더에 있는 InitDB 함수로 DB 연결 초기화

	// Gin 라우터 생성 및 라우트 설정 분리
	r := gin.Default()
	routes.SetupRoutes(r)

	// 서버 실행
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[종료] 서버 실행 오류: %v", err)
	}
}
