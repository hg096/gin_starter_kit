package main

import (
	"log"
	"os"

	"gin_starter/db"
	"gin_starter/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal("[종료] Error loading .env file")
	}

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = gin.DebugMode // 기본적으로 debug 모드를 사용
	}
	gin.SetMode(ginMode)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db.InitDB()

	r := gin.Default()
	routes.SetupRoutes(r)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[종료] 서버 실행 오류: %v", err)
	}
}
