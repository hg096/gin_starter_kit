# CLAUDE.md

이 저장소에서 Claude가 작업할 때 읽는 진입 가이드입니다.

## 먼저 읽을 문서

1. [AGENTS.md](AGENTS.md): 공통 AI 작업 지침
2. [internal/README.md](internal/README.md): 필수 코드 스타일과 상세 규칙
3. 수정 대상 기능 파일과 관련 공통 helper
4. 필요할 때 [pkg/README.md](pkg/README.md), [README.md](README.md)

사용자의 명시적 요청을 우선합니다. 코드 스타일의 기준은 `internal/README.md`이며, 이 문서의 요약을 별도 설계 규칙으로 확장하지 않습니다. 실제 코드와 문서가 다르면 코드를 확인해 판단하고, 문서에 맞추기 위한 구조 변경은 하지 않습니다.

## 현재 코드 스타일

- 기능 파일의 handler에서 입력 -> 검증 -> 업무 처리 -> DB 호출 -> 응답을 순서대로 읽게 작성합니다.
- 기능마다 handler/service/model/repository 파일이나 interface를 기계적으로 만들지 않습니다. handler는 DB helper 또는 필요한 domain repository를 직접 사용할 수 있습니다.
- 의존성 묶음은 `deps`로 이름 짓고, 같은 설정을 중복 보관하지 않습니다.
- 복잡한 세부 처리는 같은 파일의 로컬 함수로 추출할 수 있습니다. 반복 횟수는 참고이며, 업무 의미·가독성·데이터 일관성으로 판단합니다.
- 도메인 중립 유틸은 `pkg/utils`, 입력 검증은 `pkg/validator`, DB 공통 처리는 `pkg/db/database`를 확인합니다. 업무 정책·권한·감사 순서는 해당 도메인에 둡니다.
- 기본 CRUD는 `Insert/Update/Delete`, 트랜잭션은 `WithTx`와 `Tx(tx)` view를 사용합니다. 새 `InsertTx/UpdateTx/DeleteTx/CreateTx` 메서드는 만들지 않습니다.
- 커스텀 DML은 `Exec`, 스키마 보정은 `ExecSchema`를 사용합니다. `ExecUnsafe`는 호출부에 사유가 드러나는 예외에만 사용합니다.
- 트랜잭션 조회·변경은 같은 tx를 사용하고, 다음 쿼리 전에 rows를 닫습니다. 커밋 실패는 저장 여부가 불확실하므로 자동 재실행하지 않습니다.
- 동시 실행을 막아야 하는 작업은 기존 `WithNamedLockTx`를 검토합니다. 스키마 ALTER는 존재 여부를 먼저 확인합니다.
- JSON 문자열 타입, 글자 수(`MinLen/MaxLen`), 바이트 제한(`MaxBytes`)을 구분합니다. DB 장애를 조회 없음으로 바꾸지 않습니다.
- API 응답은 `pkg/response`, 비밀번호 해시는 bcrypt를 사용합니다. 토큰·인증 변경은 현재 구현과 관련 테스트를 확인합니다.

실제 예시: [회원가입](internal/domain/user/register.go), [블로그](internal/domain/blog/posts.go), [관리자 초기화](internal/domain/admin/bootstrap.go).

## 작업과 검증

사용자 변경을 보존하고 요청 범위만 수정합니다. 기존 helper와 호출부를 먼저 검색하고, 수동 수정은 `apply_patch`를 사용합니다. 동작이 유지된다면 관련 테스트도 유지합니다.

프로젝트 루트에서 실행합니다.

```text
go run ./cmd/server
go build ./cmd/server
go test ./...
go vet ./...
```

Go 파일을 수정하면 `gofmt`와 가장 작은 관련 테스트를 실행합니다. 공통 코드 변경은 전체 테스트로 확인하고, 실행하지 못한 검증은 알립니다. 문서만 수정한 경우 링크·내용·diff 검사를 수행합니다.

설치·DB 초기화·환경변수 안내는 루트 README를 사용합니다. JWT 키는 각각 정확히 32바이트입니다. 실행 안내만으로 DB 초기화 SQL을 임의 실행하지 않습니다.
