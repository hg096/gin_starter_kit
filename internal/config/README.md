# internal/config

환경변수 기반 설정을 로딩하고 검증합니다.

## 구조

```text
config.go
  Config
  ServerConfig
  DatabaseConfig
  JWTConfig
  AppConfig
  Load/Get
  IsDevelopment/IsProduction/IsAllowedOrigin
```

## 현재 설정

```go
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	App      AppConfig
}
```

```go
type ServerConfig struct {
	Port     string
	GinMode  string
	Timeout  time.Duration
	BasePath string
}
```

```go
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}
```

```go
type JWTConfig struct {
	AccessSecret      []byte
	RefreshSecret     []byte
	TokenSecret       []byte
	AccessExpireMin   int
	RefreshExpireDays int
	RefreshReuseHours int
	Issuer            string
	Audience          string
	Subject           string
}
```

```go
type AppConfig struct {
	ServiceName        string
	Environment        string
	Debug              bool
	CORSAllowedOrigins []string
}
```

## 필수 환경변수

```env
DB_NAME=gin_starter
JWT_SECRET=32-character-access-secret!!
JWT_REFRESH_SECRET=32-character-refresh-secret!
JWT_TOKEN_SECRET=32-character-token-secret!!!
JWT_ISSUER=GinStarter
JWT_AUDIENCE=gin-starter-client
JWT_SUBJECT=access
```

JWT secret 3개는 각각 정확히 32자여야 합니다.

## 선택 환경변수

```env
PORT=8080
GIN_MODE=debug
SERVER_TIMEOUT=30
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASS=
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5
JWT_EXPIRES_IN=30
JWT_EXPIRES_RE=7
SERVICE_NAME=GinStarter
CORS_ALLOWED_ORIGINS=http://localhost:8080,http://localhost:5173
```

## 사용

```go
cfg := config.Load()
dbName := cfg.Database.Database
accessExpireMin := cfg.JWT.AccessExpireMin
```

`Load()`는 singleton으로 동작합니다. 테스트에서 환경변수를 바꿔야 하면 singleton 상태에 주의합니다.

## 수정 기준

- 설정은 `Config` 하위 struct에 타입으로 추가합니다.
- 숫자 변환은 기존 `getEnvAsInt` 흐름을 사용합니다.
- 문자열 정규화가 필요하면 `utils.TrimLower`를 우선 검토합니다.
- CORS origin 판단은 `IsAllowedOrigin`에 모읍니다.
- 다른 패키지에서 `os.Getenv`를 직접 반복하지 않습니다.
- 민감 정보는 로그에 그대로 출력하지 않습니다.
