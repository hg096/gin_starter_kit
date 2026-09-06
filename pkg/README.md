# pkg/

**외부에서 import 가능한** 재사용 가능한 패키지들입니다.

## 🎯 pkg vs internal

| 구분 | pkg/ | internal/ |
|------|------|-----------|
| **외부 import** | ✅ 가능 | ❌ 불가능 |
| **용도** | 범용 유틸리티 | 비즈니스 로직 |
| **의존성** | 최소화 | 프로젝트 특화 |
| **예시** | 로거, Validator | User, Blog 도메인 |

## 📁 디렉토리 구조

```
pkg/
├── response/    # 표준 API 응답
├── validator/   # 입력 검증
├── errors/      # 에러 관리
└── logger/      # 로깅
```

---

## 📦 response/ - 표준 API 응답

### 역할
모든 API 응답을 일관된 형식으로 반환합니다.

### 표준 응답 구조

```json
{
  "success": true,
  "data": { ... },
  "error": null,
  "meta": { ... }
}
```

### 사용 예시

```go
import "gin_starter/pkg/response"

// 성공 응답
response.Success(c, gin.H{"user": user})

// 생성 성공 (201)
response.Created(c, createdItem)

// 검증 에러 (422)
response.ValidationError(c, errorMap)

// 인증 에러 (401)
response.Unauthorized(c, "인증이 필요합니다")

// 권한 에러 (403)
response.Forbidden(c, "접근 권한이 없습니다")

// Not Found (404)
response.NotFound(c, "리소스를 찾을 수 없습니다")

// 서버 에러 (500)
response.InternalError(c, "서버 오류가 발생했습니다")

// 커스텀 에러
response.Error(c, 400, "CUSTOM_ERROR", "메시지", details)
```

### 확장 방법

새로운 응답 타입 추가:

```go
// pkg/response/response.go에 추가

// TooManyRequests 429 에러
func TooManyRequests(c *gin.Context, message string) {
    Error(c, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", message)
}

// Accepted 202 응답
func Accepted(c *gin.Context, data interface{}) {
    c.JSON(http.StatusAccepted, Response{
        Success: true,
        Data:    data,
    })
}
```

---

## 🔍 validator/ - 입력 검증

### 역할
HTTP 요청의 입력값을 검증합니다.

### 기본 사용법

```go
import "gin_starter/pkg/validator"

rules := []validator.Rule{
    {
        Field:    "email",
        Label:    "이메일",
        Required: true,
        Pattern:  validator.PatternEmail,
    },
    {
        Field:  "age",
        Label:  "나이",
        Min:    18,
        Max:    120,
    },
    {
        Field:   "username",
        Label:   "사용자명",
        Required: true,
        MinLen:  3,
        MaxLen:  20,
        Pattern: validator.PatternAlphaNum,
    },
}

result := validator.Validate(c, rules)
if !result.Valid {
    response.ValidationError(c, result.GetErrorMap())
    return
}

// 검증된 값 사용
email := result.Values["email"]
```

### 사용 가능한 패턴

```go
validator.PatternEmail       // 이메일
validator.PatternNumber      // 숫자만
validator.PatternDecimal     // 소수점 포함
validator.PatternEnglish     // 영문
validator.PatternKorean      // 한글
validator.PatternKorEng      // 한글+영문
validator.PatternKorEngNum   // 한글+영문+숫자
validator.PatternAlphaNum    // 영숫자
validator.PatternSlug        // URL 슬러그 (소문자+하이픈)
validator.PatternURL         // URL
validator.PatternPhone       // 전화번호
```

### 커스텀 검증

```go
rules := []validator.Rule{
    {
        Field:    "password",
        Label:    "비밀번호",
        Required: true,
        Custom: func(value string) error {
            // 비밀번호 강도 검사
            if !hasUpperCase(value) {
                return errors.New("대문자를 포함해야 합니다")
            }
            if !hasSpecialChar(value) {
                return errors.New("특수문자를 포함해야 합니다")
            }
            return nil
        },
    },
}
```

### 확장: 새 패턴 추가

```go
// pkg/validator/validator.go에 추가

var (
    PatternCreditCard = regexp.MustCompile(`^\d{4}-?\d{4}-?\d{4}-?\d{4}$`)
    PatternZipCode    = regexp.MustCompile(`^\d{5}$`)
)
```

---

## ❌ errors/ - 에러 관리

### 역할
애플리케이션 전체의 에러를 일관되게 관리합니다.

### 기본 사용법

```go
import "gin_starter/pkg/errors"

// 에러 생성
err := errors.New("USER_NOT_FOUND", "사용자를 찾을 수 없습니다")

// 에러 래핑 (기존 에러에 컨텍스트 추가)
err := errors.Wrap(dbErr, "DATABASE_ERROR", "사용자 조회 실패")

// 미리 정의된 에러 사용
return errors.ErrUserNotFound
return errors.ErrInvalidToken
return errors.ErrUnauthorized

// 메타데이터 추가
err.WithMeta("user_id", userID)
err.WithMeta("attempt", attemptCount)
```

### 미리 정의된 에러

```go
// 일반
errors.ErrInternal
errors.ErrBadRequest
errors.ErrNotFound
errors.ErrUnauthorized
errors.ErrForbidden
errors.ErrConflict
errors.ErrValidation

// 데이터베이스
errors.ErrDatabase
errors.ErrDuplicateEntry
errors.ErrRecordNotFound

// 인증
errors.ErrInvalidToken
errors.ErrExpiredToken
errors.ErrInvalidPassword

// 사용자
errors.ErrUserNotFound
errors.ErrUserExists
errors.ErrInvalidCredentials
```

### 에러 확인

```go
if errors.Is(err, errors.ErrUserNotFound) {
    // 사용자 없음 처리
}
```

### 확장: 새 에러 추가

```go
// pkg/errors/errors.go에 추가

var (
    // 주문 관련
    ErrOrderNotFound    = New("ORDER_NOT_FOUND", "주문을 찾을 수 없습니다")
    ErrOrderCancelled   = New("ORDER_CANCELLED", "취소된 주문입니다")

    // 결제 관련
    ErrPaymentFailed    = New("PAYMENT_FAILED", "결제에 실패했습니다")
    ErrInsufficientFund = New("INSUFFICIENT_FUND", "잔액이 부족합니다")
)
```

---

## 📝 logger/ - 로깅

### 역할
구조화된 로깅을 제공합니다.

### 기본 사용법

```go
import "gin_starter/pkg/logger"

// 레벨별 로깅
logger.Debug("디버그 메시지: %s", value)
logger.Info("정보 메시지: %s", value)
logger.Warn("경고 메시지: %s", value)
logger.Error("에러 발생: %v", err)
logger.Fatal("치명적 에러: %v", err) // 프로그램 종료

// 필드와 함께 로깅
logger.WithField("user_id", userID).Info("로그인 성공")

logger.WithFields(map[string]interface{}{
    "user_id": userID,
    "ip": ip,
    "action": "login",
}).Info("사용자 활동")
```

### 로그 레벨 설정

```go
// main.go에서
import "gin_starter/pkg/logger"

logger.SetLevel(logger.INFO)
// 또는
logger.SetLevelFromString("debug") // debug, info, warn, error, fatal
```

### 확장: 파일 로깅

```go
// pkg/logger/logger.go에 추가

import "os"

func SetOutputFile(filename string) error {
    file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        return err
    }
    log.SetOutput(file)
    return nil
}
```

---

## 🚀 새 패키지 추가 가이드

### 1. 패키지 추가 기준

다음 조건을 **모두** 만족하면 pkg/에 추가:

- ✅ 다른 프로젝트에서도 사용 가능
- ✅ 비즈니스 로직과 무관
- ✅ 범용적인 기능
- ✅ 외부 의존성 최소

### 2. 추가 순서

```bash
# 1. 디렉토리 생성
mkdir pkg/cache

# 2. 파일 생성
cd pkg/cache
touch cache.go
```

### 3. 코드 작성 예시

```go
// pkg/cache/cache.go
package cache

import "time"

// Interface 정의
type Cache interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}, ttl time.Duration) error
    Delete(key string) error
}

// 메모리 캐시 구현
type memoryCache struct {
    data map[string]cacheItem
}

type cacheItem struct {
    value  interface{}
    expiry time.Time
}

func NewMemoryCache() Cache {
    return &memoryCache{
        data: make(map[string]cacheItem),
    }
}

func (c *memoryCache) Get(key string) (interface{}, bool) {
    item, exists := c.data[key]
    if !exists {
        return nil, false
    }

    if time.Now().After(item.expiry) {
        delete(c.data, key)
        return nil, false
    }

    return item.value, true
}

func (c *memoryCache) Set(key string, value interface{}, ttl time.Duration) error {
    c.data[key] = cacheItem{
        value:  value,
        expiry: time.Now().Add(ttl),
    }
    return nil
}

func (c *memoryCache) Delete(key string) error {
    delete(c.data, key)
    return nil
}
```

### 4. README 작성

```bash
touch pkg/cache/README.md
```

---

## 📚 패키지 사용 원칙

### DO ✅

1. **재사용성 우선**
   ```go
   // ✅ 좋은 예 - 범용적
   func ValidateEmail(email string) bool

   // ✅ 좋은 예 - 프로젝트 무관
   func FormatPhoneNumber(phone string) string
   ```

2. **명확한 Interface**
   ```go
   type Logger interface {
       Info(msg string)
       Error(msg string)
   }
   ```

3. **최소 의존성**
   ```go
   // ✅ 표준 라이브러리만 사용
   import "time"
   import "strings"
   ```

### DON'T ❌

1. **비즈니스 로직 포함 금지**
   ```go
   // ❌ 나쁜 예 - 비즈니스 로직
   func CalculateUserDiscount(user User) float64
   ```

2. **프로젝트 특화 코드 금지**
   ```go
   // ❌ 나쁜 예 - 특정 프로젝트에만 유효
   func GetUserFromGinContext(c *gin.Context) *User
   ```

3. **많은 의존성 금지**
   ```go
   // ❌ 나쁜 예 - 의존성 과다
   import "gin_starter/internal/domain/user"
   import "gin_starter/pkg/db/database"
   ```

---

## ✅ 체크리스트

새 pkg 패키지 추가 시:

- [ ] 다른 프로젝트에서도 사용 가능한가?
- [ ] 비즈니스 로직과 무관한가?
- [ ] 외부 의존성이 최소화되어 있는가?
- [ ] Interface를 정의했는가?
- [ ] 테스트 코드를 작성했는가?
- [ ] README를 작성했는가?
- [ ] 예제 코드를 포함했는가?
