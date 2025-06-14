package main

import (
	"log"
	"os"

	"gin_starter/db"
	"gin_starter/routes"
	"gin_starter/util/utilCore"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal("[종료] Error loading .env file")
	}

	ginMode := os.Getenv("GIN_MODE")
	if utilCore.EmptyString(ginMode) {
		ginMode = gin.DebugMode // 기본적으로 debug 모드를 사용
	}
	gin.SetMode(ginMode)

	port := os.Getenv("PORT")
	if utilCore.EmptyString(port) {
		port = "8080"
	}

	db.InitDB()

	r := gin.Default()
	// 매 요청마다 현재 호스트를 반영
	// r.Use(func(c *gin.Context) {
	// 	docs.SwaggerInfo.Host = c.Request.Host
	// 	c.Next()
	// })
	routes.SetupRoutes(r)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[종료] 서버 실행 오류: %v", err)
	}
}
