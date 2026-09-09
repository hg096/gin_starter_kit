# Gin Starter

**기능 하나의 흐름을 한곳에서 읽고 수정하는 Go/Gin 스타터 킷입니다.**

회원 인증, 관리자 권한, 감사 로그, 블로그, 실시간 채팅을 바탕으로 새 프로젝트를 시작할 수 있습니다. 개발자가 여러 파일을 오가며 코드를 머릿속에서 조립하는 부담을 줄이는 데 초점을 둡니다.

## 설계 철학

회원가입을 수정하려는 개발자는 입력이 어디서 들어오고, 어떤 검증을 거쳐, 무엇을 저장하고 응답하는지 순서대로 볼 수 있어야 합니다.

```text
요청 입력 -> 파싱·검증 -> 업무 로직 -> 데이터 처리 -> 응답
```

이 흐름을 기능 파일의 핸들러에 모읍니다. 파일은 기능을 기준으로 나누고, 단순 전달만 하는 서비스·레포지토리 계층은 기본으로 만들지 않습니다. 실제 예시는 [회원가입 코드](internal/domain/user/register.go)에서 확인할 수 있습니다.

공통 함수는 개발자가 매번 같은 세부 사항을 작성하지 않도록 돕습니다. 입력 검증, 페이지네이션, DB CRUD처럼 도메인과 무관한 처리는 공통 패키지에서 제공합니다. 권한 확인·데이터 변경·감사 기록처럼 반드시 같은 순서를 지켜야 하는 업무는 해당 도메인의 함수로 묶습니다.

- 기능의 처리 순서는 호출부에 드러냅니다.
- 반복되는 코드라도 업무 의미가 다르면 성급하게 하나로 합치지 않습니다.
- 공통화는 의도를 더 명확하게 하거나 데이터 일관성을 지킬 때 적용합니다.
- 복잡한 세부 처리는 이름 있는 함수로 추출하되, 전체 흐름은 기능 파일에서 이해할 수 있게 유지합니다.
- 파일 수와 코드 줄 수보다 변경할 때 이해해야 하는 범위를 줄이는 것을 우선합니다.

사람과 AI가 같은 기능 파일과 짧은 규칙을 읽고 작업하는 구조를 지향합니다. 파일 탐색과 문맥 복원을 줄이는 것이 목적이며, 토큰 절약이나 성능 향상을 보장한다는 뜻은 아닙니다.

AI 작업의 기본 규칙은 [internal/README.md](internal/README.md)입니다. 사용자가 다른 방향을 요청하지 않으면 이 규칙을 따릅니다.

## 시작하기

Go 1.25.1 이상, MySQL, MySQL CLI 또는 SQL 실행 도구가 필요합니다. 아래 명령은 프로젝트 루트에서 실행합니다.

### Git에서 빠르게 내려받기

Windows `cmd`에서 비어 있는 프로젝트 폴더로 이동한 뒤 다음 명령을 실행합니다. 저장소 이름의 하위 폴더를 만들지 않고 현재 폴더에 바로 내려받습니다.

```cmd
cd 시작 할 폴더 주소
git clone https://github.com/hg096/gin_starter_kit.git .
go mod download
```

`D:\codePack\go\draw_layer`에 기존 파일이 있으면 충돌할 수 있으므로, 새 프로젝트 폴더나 비어 있는 폴더에서 실행합니다.

이후 아래 DB 초기화와 환경변수 설정을 진행합니다.

### 1. 빈 DB 초기화

MySQL CLI에 접속합니다. `root`는 설치용 예시이며 사용할 DB 계정으로 바꿀 수 있습니다.

```powershell
mysql --default-character-set=utf8mb4 -h localhost -P 3306 -u root -p
```

아래 명령은 PowerShell이 아닌 **MySQL 프롬프트**에서 순서대로 실행합니다.

```sql
CREATE DATABASE gin_starter CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE gin_starter;
SOURCE table.sql;
SOURCE migrations/002_create_blog_table.sql;
SHOW TABLES;
EXIT;
```

각 SOURCE 실행에 오류가 없는지 확인합니다. GUI 도구에서는 `gin_starter` DB를 선택하고 두 파일을 같은 순서로 실행합니다.

`table.sql`은 사용자·권한·관리자·채팅 테이블과 초기 데이터를 만듭니다. `_blog`는 별도 `002` 파일에 있으므로 두 파일 모두 필요합니다. 현재 기본 SQL에는 `003`~`008`의 스키마 변경이 반영되어 있어 신규 설치 후 이 파일들을 추가 실행하지 않습니다.

**기존 DB에는 이 초기화 절차를 실행하지 않습니다.** `table.sql`에는 권한·관리자 메뉴를 삭제한 뒤 초기 데이터를 넣는 구문이 있습니다. 기존 DB 갱신은 백업 후 현재 스키마와 적용 이력을 확인하고, `migrations/`에서 아직 반영하지 않은 변경만 번호 순서대로 검토·적용합니다. 자동 마이그레이션 실행기와 버전 추적은 제공하지 않으며, 일부 ALTER는 재실행 시 중복 컬럼 오류가 발생합니다.

### 2. 환경변수 설정

`.env`가 없는 경우에만 예제를 복사합니다.

```powershell
if (!(Test-Path .env)) { Copy-Item exenv.txt .env }
```

DB 접속 정보를 실제 환경에 맞춥니다. JWT 키는 길이를 맞춘 **로컬 개발 전용 공개 예시**입니다.

```env
PORT=8080
GIN_MODE=debug
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASS=
DB_NAME=gin_starter

JWT_SECRET=0123456789abcdef0123456789abcdef
JWT_REFRESH_SECRET=abcdef0123456789abcdef0123456789
JWT_TOKEN_SECRET=89abcdef0123456789abcdef01234567
JWT_ISSUER=GinStarter
JWT_AUDIENCE=gin-starter-client
JWT_SUBJECT=access
```

JWT 키 세 개는 각각 **정확히 32바이트**여야 합니다. ASCII 문자열 32자를 사용하면 조건을 만족합니다. 배포할 때는 서로 다른 임의의 키로 교체합니다. 다음 PowerShell 명령은 32자리 16진수 키 세 개를 생성합니다.

```powershell
$rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
try {
    foreach ($name in @('JWT_SECRET', 'JWT_REFRESH_SECRET', 'JWT_TOKEN_SECRET')) {
        $bytes = New-Object byte[] 16
        $rng.GetBytes($bytes)
        $key = [BitConverter]::ToString($bytes).Replace('-', '').ToLowerInvariant()
        "$name=$key"
    }
} finally {
    $rng.Dispose()
}
```

DB 풀, 토큰 만료 시간, CORS 설정은 [exenv.txt](exenv.txt)를 참고합니다. 실제 비밀값이 있는 `.env`는 저장소에 커밋하지 않습니다.

### 3. 실행과 확인

```powershell
go mod download
go run ./cmd/server
```

템플릿·정적 파일을 찾을 수 있도록 프로젝트 루트에서 실행합니다. 기본 포트에서는 다음 주소로 확인합니다.

- `GET http://localhost:8080/health/live`: 프로세스 응답 확인
- `GET http://localhost:8080/health/ready`: DB 연결 상태 확인
- `http://localhost:8080/swagger/index.html`: API 문서
- `http://localhost:8080/admin`: 관리자 화면

최초 최고관리자는 `GET /api/admin/bootstrap/status`로 생성 가능 여부를 확인한 뒤 `POST /api/admin/bootstrap/super-admin`으로 생성합니다. 요청 필드는 `user_id`, `user_pass`, `user_name`, `user_email`입니다. 공유 환경을 열기 전에 초기 계정 설정을 마칩니다.

빌드와 검증:

```powershell
go build -o bin/server.exe ./cmd/server
go test ./...
go vet ./...
```

## 구성

```text
cmd/server/          실행 진입점
internal/api/routes/ HTTP 라우트 연결
internal/config/     환경변수 설정
internal/domain/     user, blog, admin 기능별 흐름
internal/websocket/  채팅 및 WebSocket
pkg/                 검증, DB, 응답, 미들웨어 등 공통 처리
web/                 템플릿과 정적 파일
docs/                프로젝트 안내와 Swagger 산출물
table.sql            신규 DB 기본 스키마와 초기 데이터
migrations/          블로그 스키마 및 기존 DB 변경 SQL
```

주요 API는 `/api/user`, `/api/blog`, `/api/admin`에 있으며 WebSocket 연결은 `/ws`를 사용합니다.

## 개발 규칙과 문서

입력 검증은 `pkg/validator`, 중립적인 유틸은 `pkg/utils`, DB 공통 처리는 `pkg/db/database`를 먼저 확인합니다. 기본 CRUD는 `Insert`, `Update`, `Delete`를 사용하고 트랜잭션 안에서도 같은 이름을 유지합니다.

- [내부 코드·AI 작업 규칙](internal/README.md)
- [프로젝트 가이드](docs/PROJECT_GUIDE.md)
- [공통 패키지](pkg/README.md)
- [DB 처리](pkg/db/README.md)
- [미들웨어](pkg/middleware/README.md)
