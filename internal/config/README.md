# internal/config/

애플리케이션의 **모든 설정을 중앙에서 관리**하는 패키지입니다.

## 🎯 역할

- 환경 변수(.env) 로딩
- 설정 검증
- 타입 안전한 설정 접근
- 싱글톤 패턴으로 전역 접근

## 📁 파일 구조

```
config/
└── config.go    # 설정 정의 및 로딩
```

## 🔧 현재 설정 구조

```go
type Config struct {
    Server   ServerConfig      // 서버 설정
    Database DatabaseConfig    // DB 설정
    JWT      JWTConfig        // JWT 설정
    App      AppConfig        // 앱 설정
}
```

### ServerConfig

```go
type ServerConfig struct {
    Port    string  // 포트 번호 (예: "8080")
    GinMode string  // Gin 모드 (debug, release, test)
}
```

### DatabaseConfig

```go
type DatabaseConfig struct {
    Host         string  // DB 호스트
    Port         string  // DB 포트
    User         string  // DB 유저
    Password     string  // DB 비밀번호
    Name         string  // DB 이름
    MaxOpenConns int     // 최대 연결 수
    MaxIdleConns int     // 유휴 연결 수
}
```

### JWTConfig

```go
type JWTConfig struct {
    Secret        []byte  // Access Token 서명 키
    RefreshSecret []byte  // Refresh Token 서명 키
    TokenSecret   []byte  // 토큰 암호화 키 (AES-GCM)
    ExpiresIn     int     // Access Token 만료 시간 (분)
    RefreshIn     int     // Refresh Token 만료 시간 (일)
}
```

### AppConfig

```go
type AppConfig struct {
    ServiceName string  // 서비스 이름
}
```

---

## 📖 사용 방법

### 1. 기본 사용

```go
import "gin_starter/internal/config"

func main() {
    // 설정 로드 (싱글톤)
    cfg := config.Load()

    // 사용
    fmt.Println(cfg.Server.Port)
    fmt.Println(cfg.Database.Host)
    fmt.Println(cfg.App.ServiceName)
}
```

### 2. 다른 패키지에서 사용

```go
// Service에서
func NewService(cfg *config.Config) Service {
    return &service{
        jwtSecret: cfg.JWT.Secret,
        tokenExpiry: cfg.JWT.ExpiresIn,
    }
}

// Handler에서
func NewHandler(cfg *config.Config) *Handler {
    return &Handler{
        serviceName: cfg.App.ServiceName,
    }
}
```

---

## 🚀 새 설정 추가하기

### Step 1: Config 구조체에 추가

```go
// config.go

// 새 설정 그룹 정의
type RedisConfig struct {
    Host     string
    Port     string
    Password string
    DB       int
}

// Config에 추가
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    JWT      JWTConfig
    App      AppConfig
    Redis    RedisConfig    // 추가!
}
```

### Step 2: Load 함수에서 환경변수 읽기

```go
func Load() *Config {
    once.Do(func() {
        // ...기존 코드...

        // Redis 설정 추가
        instance.Redis = RedisConfig{
            Host:     getEnv("REDIS_HOST", "localhost"),
            Port:     getEnv("REDIS_PORT", "6379"),
            Password: getEnv("REDIS_PASSWORD", ""),
            DB:       getEnvInt("REDIS_DB", 0),
        }

        // ...검증...
    })
    return instance
}
```

### Step 3: 검증 로직 추가

```go
func (c *Config) validate() error {
    // ...기존 검증...

    // Redis 검증
    if c.Redis.Host == "" {
        return fmt.Errorf("REDIS_HOST가 설정되지 않았습니다")
    }

    return nil
}
```

### Step 4: .env 파일 업데이트

```env
# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

### Step 5: 사용

```go
// infrastructure/cache/redis.go
func Connect(cfg *config.Config) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
        Password: cfg.Redis.Password,
        DB:       cfg.Redis.DB,
    })

    return client, nil
}
```

---

## 🔍 고급 예제

### 1. 환경별 설정 분리

```go
type Config struct {
    // ...기존 설정...
    Environment string  // dev, staging, prod
}

func Load() *Config {
    once.Do(func() {
        instance = &Config{}
        loadEnv()

        env := getEnv("ENVIRONMENT", "dev")
        instance.Environment = env

        // 환경별 설정
        switch env {
        case "prod":
            instance.Server.GinMode = "release"
            instance.Database.MaxOpenConns = 100
        case "staging":
            instance.Server.GinMode = "release"
            instance.Database.MaxOpenConns = 50
        default:
            instance.Server.GinMode = "debug"
            instance.Database.MaxOpenConns = 25
        }

        // 나머지 설정...
    })
    return instance
}
```

### 2. 민감 정보 마스킹

```go
func (c *Config) String() string {
    return fmt.Sprintf(
        "Config{Server:%+v, DB:%s@%s:%s/%s, JWT:***}",
        c.Server,
        c.Database.User,
        c.Database.Host,
        c.Database.Port,
        c.Database.Name,
    )
}
```

### 3. 설정 Hot Reload

```go
import "github.com/fsnotify/fsnotify"

func WatchConfig() {
    watcher, _ := fsnotify.NewWatcher()
    defer watcher.Close()

    watcher.Add(".env")

    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Write == fsnotify.Write {
                logger.Info(".env 파일이 변경되었습니다. 재로딩...")
                instance = nil  // 싱글톤 초기화
                once = sync.Once{}
                Load()
            }
        }
    }
}
```

### 4. 설정 검증 강화

```go
func (c *Config) validate() error {
    var errs []string

    // Server 검증
    if port, err := strconv.Atoi(c.Server.Port); err != nil || port < 1024 || port > 65535 {
        errs = append(errs, "PORT는 1024-65535 사이여야 합니다")
    }

    // Database 검증
    if c.Database.MaxOpenConns < c.Database.MaxIdleConns {
        errs = append(errs, "DB_MAX_OPEN_CONNS는 DB_MAX_IDLE_CONNS보다 커야 합니다")
    }

    // JWT 검증
    if len(c.JWT.Secret) != 32 {
        errs = append(errs, "JWT_SECRET은 정확히 32자여야 합니다")
    }
    if len(c.JWT.RefreshSecret) != 32 {
        errs = append(errs, "JWT_REFRESH_SECRET은 정확히 32자여야 합니다")
    }
    if len(c.JWT.TokenSecret) != 32 {
        errs = append(errs, "JWT_TOKEN_SECRET은 정확히 32자여야 합니다")
    }

    if len(errs) > 0 {
        return fmt.Errorf("설정 검증 실패:\n- %s", strings.Join(errs, "\n- "))
    }

    return nil
}
```

---

## 📚 패턴별 예시

### 외부 API 설정

```go
type ExternalAPIConfig struct {
    PaymentAPIURL    string
    PaymentAPIKey    string
    PaymentTimeout   time.Duration
    NotificationURL  string
}

func Load() *Config {
    // ...
    instance.ExternalAPI = ExternalAPIConfig{
        PaymentAPIURL:   getEnv("PAYMENT_API_URL", ""),
        PaymentAPIKey:   getEnv("PAYMENT_API_KEY", ""),
        PaymentTimeout:  time.Duration(getEnvInt("PAYMENT_TIMEOUT", 30)) * time.Second,
        NotificationURL: getEnv("NOTIFICATION_URL", ""),
    }
    // ...
}
```

### 파일 업로드 설정

```go
type FileUploadConfig struct {
    MaxFileSize   int64   // bytes
    AllowedTypes  []string
    UploadPath    string
}

func Load() *Config {
    // ...
    maxSize, _ := strconv.ParseInt(getEnv("MAX_FILE_SIZE", "10485760"), 10, 64) // 10MB

    instance.FileUpload = FileUploadConfig{
        MaxFileSize:  maxSize,
        AllowedTypes: strings.Split(getEnv("ALLOWED_FILE_TYPES", "jpg,png,pdf"), ","),
        UploadPath:   getEnv("UPLOAD_PATH", "./uploads"),
    }
    // ...
}
```

### 이메일 설정

```go
type EmailConfig struct {
    SMTPHost     string
    SMTPPort     int
    SMTPUser     string
    SMTPPassword string
    FromAddress  string
    FromName     string
}

func Load() *Config {
    // ...
    instance.Email = EmailConfig{
        SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
        SMTPPort:     getEnvInt("SMTP_PORT", 587),
        SMTPUser:     getEnv("SMTP_USER", ""),
        SMTPPassword: getEnv("SMTP_PASSWORD", ""),
        FromAddress:  getEnv("EMAIL_FROM", "noreply@example.com"),
        FromName:     getEnv("EMAIL_FROM_NAME", "GinStarter"),
    }
    // ...
}
```

---

## ✅ 체크리스트

새 설정 추가 시:

- [ ] Config 구조체에 타입 정의
- [ ] Load() 함수에서 환경변수 읽기
- [ ] 기본값(default) 설정
- [ ] validate() 함수에 검증 로직 추가
- [ ] .env.example 파일 업데이트
- [ ] README.md 업데이트
- [ ] 민감 정보는 String() 메서드에서 마스킹

---

## 💡 팁

### 1. 타입 변환 헬퍼

```go
// getEnvInt 정수 환경변수
func getEnvInt(key string, defaultVal int) int {
    if val := os.Getenv(key); val != "" {
        if intVal, err := strconv.Atoi(val); err == nil {
            return intVal
        }
    }
    return defaultVal
}

// getEnvBool 불린 환경변수
func getEnvBool(key string, defaultVal bool) bool {
    if val := os.Getenv(key); val != "" {
        if boolVal, err := strconv.ParseBool(val); err == nil {
            return boolVal
        }
    }
    return defaultVal
}

// getEnvDuration 기간 환경변수
func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
    if val := os.Getenv(key); val != "" {
        if duration, err := time.ParseDuration(val); err == nil {
            return duration
        }
    }
    return defaultVal
}
```

### 2. 설정 로깅

```go
func Load() *Config {
    // ...로딩...

    // 설정 로깅 (민감 정보 제외)
    logger.Info("설정 로드 완료: %s", instance.String())

    return instance
}
```

### 3. 필수 설정 강제

```go
func getEnvRequired(key string) string {
    val := os.Getenv(key)
    if val == "" {
        logger.Fatal("%s 환경변수가 설정되지 않았습니다", key)
    }
    return val
}

// 사용
instance.Database.Password = getEnvRequired("DB_PASS")
```

---

## ⚠️ 주의사항

### DO ✅

- 모든 설정은 config 패키지를 통해 접근
- 민감 정보는 환경변수로만 관리
- 기본값을 항상 제공
- 설정 검증 로직 작성

### DON'T ❌

- 코드에 비밀번호/키 하드코딩 금지
- 여러 곳에서 os.Getenv() 직접 호출 금지
- 전역 변수로 설정 관리 금지 (싱글톤 사용)
- .env 파일을 git에 커밋 금지

---

## 📚 참고

- [12 Factor App - Config](https://12factor.net/config)
- [godotenv](https://github.com/joho/godotenv)