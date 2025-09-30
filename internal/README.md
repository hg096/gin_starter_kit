# internal/

`internal` 패키지는 **외부에서 import 불가능**한 애플리케이션의 핵심 비즈니스 로직을 포함합니다.

## 📁 디렉토리 구조

```
internal/
├── config/          # 설정 관리
├── domain/          # 비즈니스 도메인
├── middleware/      # HTTP 미들웨어
└── infrastructure/  # 외부 시스템 연동
```

## 🎯 핵심 원칙

### 1. 외부 Import 금지
`internal/` 패키지는 Go의 특수 디렉토리로, 같은 모듈 내에서만 import 가능합니다.

```go
// ✅ 같은 프로젝트 내에서 OK
import "gin_starter/internal/domain/user"

// ❌ 다른 프로젝트에서 불가능
import "github.com/other/project/internal/domain/user" // 컴파일 에러
```

### 2. 비즈니스 로직 중심
외부 라이브러리나 프레임워크에 의존하지 않는 **순수 비즈니스 로직**을 작성합니다.

### 3. 의존성 방향
```
domain (핵심)
  ↑
  ├── infrastructure (구현)
  ├── middleware (지원)
  └── config (설정)
```

## 📖 각 디렉토리 역할

### config/
전역 설정 관리. 환경변수, 데이터베이스 설정, JWT 설정 등 모든 설정을 중앙화합니다.

**특징:**
- 싱글톤 패턴
- 환경별 설정 분리 (dev, staging, prod)
- 타입 안전성

### domain/
**가장 중요한 디렉토리**. 각 비즈니스 도메인별로 독립적인 폴더를 생성합니다.

**특징:**
- 완전한 독립성 (다른 도메인에 의존 X)
- 표준 구조: model → repository → service → handler
- Interface 기반 설계

### middleware/
HTTP 요청/응답 처리 전후에 실행되는 공통 로직입니다.

**특징:**
- 횡단 관심사 (인증, 로깅, CORS)
- 재사용 가능
- 순서 중요

### infrastructure/
외부 시스템 연동 코드입니다. 데이터베이스, 캐시, 메시지 큐 등

**특징:**
- 외부 의존성 격리
- Interface로 추상화
- 교체 가능한 구조

## 🚀 새 기능 추가 가이드

### 1. 새 도메인 추가 (예: Product)

```bash
mkdir -p internal/domain/product
cd internal/domain/product
touch model.go repository.go service.go handler.go
```

### 2. 새 미들웨어 추가

```bash
cd internal/middleware
touch ratelimit.go
```

### 3. 새 Infrastructure 추가 (예: Redis)

```bash
mkdir -p internal/infrastructure/cache
cd internal/infrastructure/cache
touch redis.go
```

## ⚠️ 주의사항

### DO ✅
- 각 패키지는 명확한 단일 책임
- Interface 우선 설계
- 에러는 항상 래핑
- 중요한 작업은 로깅

### DON'T ❌
- domain 간 직접 의존 금지
- infrastructure를 domain에서 직접 import 금지
- 전역 변수 사용 금지 (config 제외)
- 비즈니스 로직을 middleware에 작성 금지

## 📚 참고

- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Dependency Inversion Principle](https://en.wikipedia.org/wiki/Dependency_inversion_principle)