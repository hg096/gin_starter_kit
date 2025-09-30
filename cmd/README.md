# cmd/

애플리케이션의 **진입점(Entry Point)**을 관리하는 디렉토리입니다.

## 🎯 역할

- 애플리케이션 시작점 정의
- 의존성 초기화 및 연결
- Graceful Shutdown 처리
- 환경 설정 로드

## 📁 디렉토리 구조

```
cmd/
└── server/
    └── main.go    # 웹 서버 진입점
```

---

## 🚀 현재 구조

### server/main.go

**역할**: HTTP 서버 실행

**주요 기능**:
1. 설정 로드
2. 데이터베이스 연결
3. 라우터 설정
4. 서버 시작
5. Graceful Shutdown

**실행 방법**:
```bash
# 개발 모드
go run cmd/server/main.go

# 빌드 후 실행
go build -o bin/server cmd/server/main.go
./bin/server
```

---

## 📖 main.go 구조 분석

### 1. 기본 구조

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "gin_starter/api/routes"
    "gin_starter/internal/config"
    "gin_starter/internal/infrastructure/database"
    "gin_starter/pkg/logger"

    "github.com/gin-gonic/gin"
)

func main() {
    // 1. 설정 로드
    cfg := config.Load()

    // 2. 데이터베이스 연결
    db, err := database.Connect(cfg)
    if err != nil {
        logger.Fatal("DB 연결 실패: %v", err)
    }
    defer db.Close()

    // 3. Gin 라우터 설정
    r := gin.New()
    routes.SetupRoutes(r, db, cfg)

    // 4. HTTP 서버 생성
    srv := &http.Server{
        Addr:    ":" + cfg.Server.Port,
        Handler: r,
    }

    // 5. Graceful Shutdown
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("서버 시작 실패: %v", err)
        }
    }()

    logger.Info("서버 시작: http://localhost:%s", cfg.Server.Port)

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("서버 종료 중...")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        logger.Fatal("서버 강제 종료: %v", err)
    }

    logger.Info("서버 정상 종료")
}
```

### 2. 각 단계 설명

#### 1단계: 설정 로드
```go
cfg := config.Load()
```
- `.env` 파일에서 환경변수 읽기
- 설정 검증
- 싱글톤 패턴으로 전역 접근 가능

#### 2단계: 데이터베이스 연결
```go
db, err := database.Connect(cfg)
if err != nil {
    logger.Fatal("DB 연결 실패: %v", err)
}
defer db.Close()
```
- MySQL 연결 및 연결 풀 설정
- 연결 실패 시 프로그램 종료
- `defer`로 프로그램 종료 시 연결 닫기

#### 3단계: 라우터 설정
```go
r := gin.New()
routes.SetupRoutes(r, db, cfg)
```
- Gin 엔진 생성
- 모든 라우트와 미들웨어 등록

#### 4단계: HTTP 서버 생성
```go
srv := &http.Server{
    Addr:    ":" + cfg.Server.Port,
    Handler: r,
}
```
- 표준 HTTP 서버 생성
- Graceful Shutdown을 위해 `http.Server` 사용

#### 5단계: Graceful Shutdown
```go
go func() {
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Fatal("서버 시작 실패: %v", err)
    }
}()

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := srv.Shutdown(ctx); err != nil {
    logger.Fatal("서버 강제 종료: %v", err)
}
```
- SIGINT (Ctrl+C), SIGTERM 신호 대기
- 신호 수신 시 진행 중인 요청 완료 후 종료
- 5초 타임아웃 설정

---

## 🔧 새 진입점 추가하기

### 1. Worker 진입점 추가

#### Step 1: 디렉토리 생성

```bash
mkdir -p cmd/worker
cd cmd/worker
touch main.go
```

#### Step 2: Worker 코드 작성

```go
// cmd/worker/main.go
package main

import (
    "os"
    "os/signal"
    "syscall"

    "gin_starter/internal/config"
    "gin_starter/internal/infrastructure/database"
    "gin_starter/internal/infrastructure/queue"
    "gin_starter/pkg/logger"
)

func main() {
    // 1. 설정 로드
    cfg := config.Load()

    // 2. 데이터베이스 연결
    db, err := database.Connect(cfg)
    if err != nil {
        logger.Fatal("DB 연결 실패: %v", err)
    }
    defer db.Close()

    // 3. 메시지 큐 연결
    mq, err := queue.Connect(cfg)
    if err != nil {
        logger.Fatal("Queue 연결 실패: %v", err)
    }
    defer mq.Close()

    logger.Info("Worker 시작")

    // 4. 메시지 소비 시작
    startWorkers(mq, db)

    // 5. 종료 대기
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Worker 종료")
}

func startWorkers(mq *queue.RabbitMQ, db *database.DB) {
    // 이메일 발송 워커
    mq.Consume("email-queue", func(data []byte) error {
        // 이메일 발송 로직
        logger.Info("이메일 발송: %s", string(data))
        return nil
    })

    // 알림 발송 워커
    mq.Consume("notification-queue", func(data []byte) error {
        // 알림 발송 로직
        logger.Info("알림 발송: %s", string(data))
        return nil
    })
}
```

#### Step 3: 실행

```bash
# 개발 모드
go run cmd/worker/main.go

# 빌드
go build -o bin/worker cmd/worker/main.go
./bin/worker
```

---

### 2. CLI 도구 추가

```bash
mkdir -p cmd/cli
cd cmd/cli
touch main.go
```

```go
// cmd/cli/main.go
package main

import (
    "flag"
    "fmt"
    "os"

    "gin_starter/internal/config"
    "gin_starter/internal/domain/user"
    "gin_starter/internal/infrastructure/database"
    "gin_starter/pkg/logger"
)

func main() {
    // 서브 커맨드 정의
    createCmd := flag.NewFlagSet("create-user", flag.ExitOnError)
    userID := createCmd.String("id", "", "사용자 ID")
    userName := createCmd.String("name", "", "사용자 이름")
    userEmail := createCmd.String("email", "", "사용자 이메일")

    if len(os.Args) < 2 {
        fmt.Println("사용법: cli [command]")
        fmt.Println("Commands:")
        fmt.Println("  create-user  새 사용자 생성")
        os.Exit(1)
    }

    cfg := config.Load()
    db, err := database.Connect(cfg)
    if err != nil {
        logger.Fatal("DB 연결 실패: %v", err)
    }
    defer db.Close()

    switch os.Args[1] {
    case "create-user":
        createCmd.Parse(os.Args[2:])
        if *userID == "" || *userName == "" || *userEmail == "" {
            createCmd.PrintDefaults()
            os.Exit(1)
        }

        repo := user.NewRepository(db)
        service := user.NewService(repo, cfg)

        req := &user.CreateUserRequest{
            ID:       *userID,
            Password: "defaultpass",
            Name:     *userName,
            Email:    *userEmail,
        }

        if _, err := service.Register(req); err != nil {
            logger.Fatal("사용자 생성 실패: %v", err)
        }

        logger.Info("사용자 생성 완료: %s", *userID)

    default:
        fmt.Printf("알 수 없는 명령: %s\n", os.Args[1])
        os.Exit(1)
    }
}
```

**사용:**
```bash
go run cmd/cli/main.go create-user --id=admin --name=관리자 --email=admin@example.com
```

---

### 3. Migration 도구

```bash
mkdir -p cmd/migrate
cd cmd/migrate
touch main.go
```

```go
// cmd/migrate/main.go
package main

import (
    "flag"
    "fmt"
    "os"

    "gin_starter/internal/config"
    "gin_starter/internal/infrastructure/database"
    "gin_starter/pkg/logger"
)

func main() {
    action := flag.String("action", "", "up 또는 down")
    flag.Parse()

    if *action == "" {
        fmt.Println("사용법: migrate -action=[up|down]")
        os.Exit(1)
    }

    cfg := config.Load()
    db, err := database.Connect(cfg)
    if err != nil {
        logger.Fatal("DB 연결 실패: %v", err)
    }
    defer db.Close()

    switch *action {
    case "up":
        if err := migrateUp(db); err != nil {
            logger.Fatal("Migration 실패: %v", err)
        }
        logger.Info("Migration 완료")

    case "down":
        if err := migrateDown(db); err != nil {
            logger.Fatal("Rollback 실패: %v", err)
        }
        logger.Info("Rollback 완료")

    default:
        fmt.Printf("알 수 없는 액션: %s\n", *action)
        os.Exit(1)
    }
}

func migrateUp(db *database.DB) error {
    migrations := []string{
        `CREATE TABLE IF NOT EXISTS _blog (
            id INT AUTO_INCREMENT PRIMARY KEY,
            title VARCHAR(255) NOT NULL,
            content TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
    }

    for _, sql := range migrations {
        if _, err := db.DB.Exec(sql); err != nil {
            return err
        }
    }
    return nil
}

func migrateDown(db *database.DB) error {
    rollbacks := []string{
        `DROP TABLE IF EXISTS _blog`,
    }

    for _, sql := range rollbacks {
        if _, err := db.DB.Exec(sql); err != nil {
            return err
        }
    }
    return nil
}
```

**사용:**
```bash
go run cmd/migrate/main.go -action=up
go run cmd/migrate/main.go -action=down
```

---

## 📦 빌드 및 배포

### 단일 바이너리 빌드

```bash
# 현재 플랫폼
go build -o bin/server cmd/server/main.go

# Linux
GOOS=linux GOARCH=amd64 go build -o bin/server-linux cmd/server/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/server.exe cmd/server/main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o bin/server-mac cmd/server/main.go
```

### 크기 최적화

```bash
# 디버그 정보 제거, 최적화
go build -ldflags="-s -w" -o bin/server cmd/server/main.go

# UPX로 추가 압축 (선택)
upx --best --lzma bin/server
```

### Makefile 작성

```makefile
# Makefile
.PHONY: build run clean test

# 빌드
build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/worker cmd/worker/main.go
	go build -o bin/cli cmd/cli/main.go

# 실행
run:
	go run cmd/server/main.go

# 클린
clean:
	rm -rf bin/

# 테스트
test:
	go test ./...

# Swagger 생성
swagger:
	swag init -g cmd/server/main.go

# 의존성 설치
deps:
	go mod download
	go mod tidy
```

---

## 🐳 Docker 지원

### Dockerfile

```dockerfile
# Build stage
FROM golang:1.25.1-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -ldflags="-s -w" -o server cmd/server/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/server .
COPY .env .

EXPOSE 8080

CMD ["./server"]
```

### docker-compose.yml

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=mysql
      - DB_PORT=3306
    depends_on:
      - mysql
    restart: unless-stopped

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpass
      MYSQL_DATABASE: gin_starter
    volumes:
      - mysql_data:/var/lib/mysql
    ports:
      - "3306:3306"

volumes:
  mysql_data:
```

**실행:**
```bash
docker-compose up -d
```

---

## ✅ 체크리스트

새 진입점 추가 시:

- [ ] cmd/ 하위에 디렉토리 생성
- [ ] main.go 작성
- [ ] 설정 로드
- [ ] 필요한 인프라 연결
- [ ] Graceful Shutdown 구현
- [ ] 에러 로깅
- [ ] README 업데이트
- [ ] Makefile 업데이트

---

## 💡 팁

### 1. 환경별 실행

```go
func main() {
    env := os.Getenv("ENVIRONMENT")
    if env == "" {
        env = "development"
    }

    switch env {
    case "production":
        gin.SetMode(gin.ReleaseMode)
    case "test":
        gin.SetMode(gin.TestMode)
    default:
        gin.SetMode(gin.DebugMode)
    }

    // ...
}
```

### 2. 버전 정보 표시

```go
var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)

func main() {
    logger.Info("버전: %s (빌드: %s, 커밋: %s)", Version, BuildTime, GitCommit)
    // ...
}
```

**빌드 시 주입:**
```bash
go build -ldflags="-X main.Version=1.0.0 -X main.BuildTime=$(date +%Y%m%d-%H%M%S) -X main.GitCommit=$(git rev-parse HEAD)" -o bin/server cmd/server/main.go
```

---

## 📚 참고

- [Go Build Constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [Graceful Shutdown](https://gin-gonic.com/docs/examples/graceful-restart-or-stop/)
- [Cross Compilation](https://go.dev/doc/install/source#environment)