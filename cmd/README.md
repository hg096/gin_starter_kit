# cmd

애플리케이션 실행 진입점을 둡니다.

## 구조

```text
cmd/
  server/
    main.go
```

현재 실제 진입점은 `cmd/server/main.go`입니다.

## server/main.go 역할

```text
config.Load
-> database.Connect
-> gin.New
-> routes.SetupRoutes
-> http.Server 실행
-> graceful shutdown
```

## 실행

```powershell
go run ./cmd/server
```

빌드:

```powershell
go build -o bin/server.exe ./cmd/server
```

## 수정 기준

- `main.go`에는 애플리케이션 조립과 실행만 둡니다.
- route 연결은 `internal/api/routes`에서 처리합니다.
- 비즈니스 로직은 `internal/domain`에 둡니다.
- DB 작업은 domain repository 또는 `pkg/db/database` helper를 통해 처리합니다.
- 새 worker/cli/migrate 진입점은 실제 요구가 있을 때만 추가합니다.
- 새 진입점을 추가하면 이 README와 루트 README를 함께 갱신합니다.

## 테스트

진입점 조립이나 route wiring이 바뀌면:

```powershell
go test ./internal/api/routes
go test ./...
```
