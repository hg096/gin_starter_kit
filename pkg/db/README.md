# pkg/db/

**외부 시스템과의 연동**을 담당하는 레이어입니다. 데이터베이스, 캐시, 메시지 큐 등 모든 외부 의존성을 격리합니다.

## 🎯 역할

- 외부 시스템 연결 및 관리
- 도메인 로직과 외부 의존성 분리
- Interface로 추상화하여 교체 가능한 구조
- 에러 처리 및 재시도 로직

## 📁 디렉토리 구조

```
infrastructure/
└── database/
    ├── mysql.go       # MySQL 연결
    └── repository.go  # 공통 쿼리 함수
```

---

## 💾 database/ - 데이터베이스

### mysql.go - 연결 관리

#### 기능
- MySQL 연결 및 연결 풀 관리
- Health Check
- Transaction 관리

#### 사용 예시

```go
import "gin_starter/pkg/db/database"

// 연결
db, err := database.Connect(cfg)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Health Check
if err := db.Ping(); err != nil {
    log.Fatal("DB 연결 실패")
}

// Transaction 시작
tx, err := db.BeginTx()
if err != nil {
    return err
}
defer tx.Rollback()  // Commit 실패 시 자동 롤백

// 트랜잭션 작업...
if err := someOperation(tx); err != nil {
    return err  // 자동 롤백
}

// 성공 시 커밋
tx.Commit()
```

### repository.go - 공통 쿼리

#### 기능
공통 CRUD 함수 제공으로 **코드 중복 제거**

- Insert: 데이터 삽입
- Update: 데이터 수정
- Delete: 데이터 삭제
- Query: 복잡한 조회
- Exists: 존재 여부 확인
- Count: 개수 세기

#### 사용 예시

```go
// Repository 생성
repo := database.NewRepository(db)

// INSERT
data := map[string]interface{}{
    "user_id": "testuser",
    "user_name": "홍길동",
}
id, err := repo.Insert("_user", data)

// UPDATE
updates := map[string]interface{}{
    "user_name": "김철수",
}
affected, err := repo.Update("_user", updates, "user_id = ?", "testuser")

// DELETE
affected, err := repo.Delete("_user", "user_id = ?", "testuser")

// EXISTS
exists, err := repo.Exists("_user", "user_email = ?", "test@example.com")

// COUNT
count, err := repo.Count("_user", "auth_type = ?", "U")
```

---

## 🚀 새 Infrastructure 추가 가이드

### 1. Redis 캐시 추가

#### Step 1: 디렉토리 및 파일 생성

```bash
mkdir -p pkg/db/cache
cd pkg/db/cache
touch redis.go
```

#### Step 2: 연결 코드 작성

```go
// pkg/db/cache/redis.go
package cache

import (
    "context"
    "fmt"
    "time"

    "gin_starter/internal/config"
    "gin_starter/pkg/logger"

    "github.com/redis/go-redis/v9"
)

type Redis struct {
    client *redis.Client
    ctx    context.Context
}

// Connect Redis 연결
func Connect(cfg *config.Config) (*Redis, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
        Password: cfg.Redis.Password,
        DB:       cfg.Redis.DB,
    })

    // 연결 테스트
    ctx := context.Background()
    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("Redis 연결 실패: %w", err)
    }

    logger.Info("Redis 연결 성공: %s:%s", cfg.Redis.Host, cfg.Redis.Port)

    return &Redis{
        client: client,
        ctx:    ctx,
    }, nil
}

// Close 연결 종료
func (r *Redis) Close() error {
    return r.client.Close()
}

// Set 값 저장
func (r *Redis) Set(key string, value interface{}, expiration time.Duration) error {
    return r.client.Set(r.ctx, key, value, expiration).Err()
}

// Get 값 조회
func (r *Redis) Get(key string) (string, error) {
    return r.client.Get(r.ctx, key).Result()
}

// Delete 값 삭제
func (r *Redis) Delete(key string) error {
    return r.client.Del(r.ctx, key).Err()
}

// Exists 존재 여부
func (r *Redis) Exists(key string) (bool, error) {
    result, err := r.client.Exists(r.ctx, key).Result()
    return result > 0, err
}
```

#### Step 3: Config 추가

```go
// internal/config/config.go에 추가

type RedisConfig struct {
    Host     string
    Port     string
    Password string
    DB       int
}

type Config struct {
    // ...기존 설정...
    Redis RedisConfig
}
```

#### Step 4: main.go에서 초기화

```go
// cmd/server/main.go

func main() {
    cfg := config.Load()

    // Database
    db, err := database.Connect(cfg)
    if err != nil {
        logger.Fatal("DB 연결 실패: %v", err)
    }
    defer db.Close()

    // Redis 추가
    redis, err := cache.Connect(cfg)
    if err != nil {
        logger.Fatal("Redis 연결 실패: %v", err)
    }
    defer redis.Close()

    // 라우트 설정 (Redis 전달)
    routes.SetupRoutes(r, db, redis, cfg)
}
```

#### Step 5: 사용

```go
// Service에서 Redis 사용
type service struct {
    repo  Repository
    redis *cache.Redis
}

func (s *service) GetUser(id string) (*User, error) {
    // 1. 캐시 확인
    cacheKey := fmt.Sprintf("user:%s", id)
    if cached, err := s.redis.Get(cacheKey); err == nil {
        var user User
        json.Unmarshal([]byte(cached), &user)
        return &user, nil
    }

    // 2. DB 조회
    user, err := s.repo.FindByID(id)
    if err != nil {
        return nil, err
    }

    // 3. 캐시 저장
    userJSON, _ := json.Marshal(user)
    s.redis.Set(cacheKey, userJSON, 10*time.Minute)

    return user, nil
}
```

---

### 2. 메시지 큐 (RabbitMQ) 추가

#### Step 1: 파일 생성

```bash
mkdir -p pkg/db/queue
cd pkg/db/queue
touch rabbitmq.go
```

#### Step 2: 연결 코드

```go
// pkg/db/queue/rabbitmq.go
package queue

import (
    "fmt"

    "gin_starter/internal/config"
    "gin_starter/pkg/logger"

    amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
    conn    *amqp.Connection
    channel *amqp.Channel
}

func Connect(cfg *config.Config) (*RabbitMQ, error) {
    url := fmt.Sprintf("amqp://%s:%s@%s:%s/",
        cfg.RabbitMQ.User,
        cfg.RabbitMQ.Password,
        cfg.RabbitMQ.Host,
        cfg.RabbitMQ.Port,
    )

    conn, err := amqp.Dial(url)
    if err != nil {
        return nil, fmt.Errorf("RabbitMQ 연결 실패: %w", err)
    }

    channel, err := conn.Channel()
    if err != nil {
        conn.Close()
        return nil, fmt.Errorf("Channel 생성 실패: %w", err)
    }

    logger.Info("RabbitMQ 연결 성공")

    return &RabbitMQ{
        conn:    conn,
        channel: channel,
    }, nil
}

func (r *RabbitMQ) Close() error {
    if err := r.channel.Close(); err != nil {
        return err
    }
    return r.conn.Close()
}

// Publish 메시지 발행
func (r *RabbitMQ) Publish(queueName string, message []byte) error {
    // 큐 선언
    _, err := r.channel.QueueDeclare(
        queueName,
        true,  // durable
        false, // autoDelete
        false, // exclusive
        false, // noWait
        nil,   // args
    )
    if err != nil {
        return err
    }

    // 메시지 발행
    return r.channel.Publish(
        "",        // exchange
        queueName, // routing key
        false,     // mandatory
        false,     // immediate
        amqp.Publishing{
            ContentType: "application/json",
            Body:        message,
        },
    )
}

// Consume 메시지 소비
func (r *RabbitMQ) Consume(queueName string, handler func([]byte) error) error {
    msgs, err := r.channel.Consume(
        queueName,
        "",    // consumer
        false, // autoAck
        false, // exclusive
        false, // noLocal
        false, // noWait
        nil,   // args
    )
    if err != nil {
        return err
    }

    go func() {
        for msg := range msgs {
            if err := handler(msg.Body); err != nil {
                logger.Error("메시지 처리 실패: %v", err)
                msg.Nack(false, true) // 재시도
            } else {
                msg.Ack(false) // 성공
            }
        }
    }()

    return nil
}
```

#### Step 3: 사용 예시

```go
// 이메일 발송 큐
type EmailMessage struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}

// 메시지 발행
func (s *service) SendEmailAsync(to, subject, body string) error {
    msg := EmailMessage{To: to, Subject: subject, Body: body}
    msgJSON, _ := json.Marshal(msg)

    return s.queue.Publish("email-queue", msgJSON)
}

// Worker에서 메시지 소비
func StartEmailWorker(queue *queue.RabbitMQ) {
    queue.Consume("email-queue", func(data []byte) error {
        var msg EmailMessage
        if err := json.Unmarshal(data, &msg); err != nil {
            return err
        }

        // 실제 이메일 발송
        return sendEmail(msg.To, msg.Subject, msg.Body)
    })
}
```

---

### 3. S3 파일 스토리지 추가

```bash
mkdir -p pkg/db/storage
cd pkg/db/storage
touch s3.go
```

```go
// pkg/db/storage/s3.go
package storage

import (
    "bytes"
    "fmt"
    "io"

    "gin_starter/internal/config"
    "gin_starter/pkg/logger"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
)

type S3Storage struct {
    client *s3.S3
    bucket string
}

func Connect(cfg *config.Config) (*S3Storage, error) {
    sess, err := session.NewSession(&aws.Config{
        Region: aws.String(cfg.AWS.Region),
        Credentials: credentials.NewStaticCredentials(
            cfg.AWS.AccessKey,
            cfg.AWS.SecretKey,
            "",
        ),
    })
    if err != nil {
        return nil, err
    }

    logger.Info("S3 연결 성공")

    return &S3Storage{
        client: s3.New(sess),
        bucket: cfg.AWS.S3Bucket,
    }, nil
}

// Upload 파일 업로드
func (s *S3Storage) Upload(key string, data []byte, contentType string) (string, error) {
    _, err := s.client.PutObject(&s3.PutObjectInput{
        Bucket:      aws.String(s.bucket),
        Key:         aws.String(key),
        Body:        bytes.NewReader(data),
        ContentType: aws.String(contentType),
    })
    if err != nil {
        return "", err
    }

    url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, key)
    return url, nil
}

// Download 파일 다운로드
func (s *S3Storage) Download(key string) ([]byte, error) {
    result, err := s.client.GetObject(&s3.GetObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return nil, err
    }
    defer result.Body.Close()

    return io.ReadAll(result.Body)
}

// Delete 파일 삭제
func (s *S3Storage) Delete(key string) error {
    _, err := s.client.DeleteObject(&s3.DeleteObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    return err
}
```

---

## 🔄 Infrastructure 패턴

### Interface 정의

모든 Infrastructure는 Interface로 추상화하여 교체 가능하게:

```go
// pkg/db/cache/cache.go
package cache

type Cache interface {
    Set(key string, value interface{}, expiration time.Duration) error
    Get(key string) (string, error)
    Delete(key string) error
    Exists(key string) (bool, error)
}

// Redis 구현
type redisCache struct { /*...*/ }

func (r *redisCache) Set(...) error { /*...*/ }

// Memory 구현 (테스트용)
type memoryCache struct { /*...*/ }

func (m *memoryCache) Set(...) error { /*...*/ }
```

### 의존성 주입

```go
// Service에서 Interface 사용
type service struct {
    repo  Repository
    cache cache.Cache  // Interface
}

// 실제 구현체는 외부에서 주입
func NewService(repo Repository, cache cache.Cache) Service {
    return &service{
        repo:  repo,
        cache: cache,
    }
}
```

---

## ✅ 체크리스트

새 Infrastructure 추가 시:

- [ ] Interface 정의
- [ ] Connect() 함수 구현 (연결 초기화)
- [ ] Close() 함수 구현 (리소스 정리)
- [ ] 에러 처리 및 로깅
- [ ] Health Check / Ping 구현
- [ ] Config 구조체 추가
- [ ] main.go에서 초기화 및 정리
- [ ] README 업데이트
- [ ] 재시도 로직 (필요시)
- [ ] 테스트 코드 작성

---

## 💡 팁

### 1. 연결 재시도

```go
func ConnectWithRetry(cfg *config.Config, maxRetries int) (*Redis, error) {
    var err error
    for i := 0; i < maxRetries; i++ {
        redis, err := Connect(cfg)
        if err == nil {
            return redis, nil
        }

        logger.Warn("Redis 연결 실패 (%d/%d): %v", i+1, maxRetries, err)
        time.Sleep(time.Second * time.Duration(i+1))
    }
    return nil, fmt.Errorf("최대 재시도 횟수 초과: %w", err)
}
```

### 2. Connection Pool

```go
type DBPool struct {
    master *sql.DB
    slaves []*sql.DB
}

func (p *DBPool) GetReadDB() *sql.DB {
    // 라운드 로빈으로 slave 선택
    return p.slaves[rand.Intn(len(p.slaves))]
}

func (p *DBPool) GetWriteDB() *sql.DB {
    return p.master
}
```

### 3. Circuit Breaker

```go
import "github.com/sony/gobreaker"

type SafeRedis struct {
    redis   *Redis
    breaker *gobreaker.CircuitBreaker
}

func (s *SafeRedis) Get(key string) (string, error) {
    result, err := s.breaker.Execute(func() (interface{}, error) {
        return s.redis.Get(key)
    })
    if err != nil {
        return "", err
    }
    return result.(string), nil
}
```

---

## 📚 참고

- [Go Database/SQL Tutorial](https://go.dev/doc/database/)
- [Redis Go Client](https://redis.uptrace.dev/)
- [AWS SDK for Go](https://aws.github.io/aws-sdk-go-v2/docs/)
- [RabbitMQ Go Client](https://github.com/rabbitmq/amqp091-go)
