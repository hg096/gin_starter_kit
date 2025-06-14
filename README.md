# Gin Starter Kit

이 프로젝트는 Go 언어의 고성능 웹 프레임워크인 **[Gin](https://github.com/gin-gonic/gin)**을 기반으로 한 스타터 킷입니다.
초기 웹 API 서버 구축 시 필요한 기본 구조와 설정을 포함하고 있어 빠르게 개발을 시작할 수 있습니다.

## 🛠️ 주요 구성 요소

- **Gin 프레임워크**: 경량이면서 빠른 라우팅 처리 지원
- **라우터 분리 구조**: 기능별로 route 파일 분리
- **환경 변수 설정 지원 (`.env`)**: 설정값을 외부에서 관리 가능
- **모듈 관리 (`go.mod`)**: 의존성 명확하게 관리
- **핸들러 및 미들웨어 기본 예시** 포함
- **구조화된 디렉토리 구성**: 유지보수가 쉽고 확장 가능한 구조
---

## 🚀 시작 방법

> ⚠️ **주의: 꼭 `exenv.txt` 파일을 참고하여 `.env` 파일을 생성하세요!**
>
> `.env` 파일이 없으면 서버 실행에 필요한 환경 변수가 누락되어 오류가 발생할 수 있습니다.


```bash
# 의존성 설치
go mod tidy

# 서버 실행
go run main.go

# docs/ 폴더가 자동 생성됨
swag init

# 스웨거가 보이지 않을때 && 갱신
swag init -g routes/routes.go

# 프로젝트를 복사해서 시작할 경우
git init
git remote add origin 프로젝트깃주소
git remote add starter-kit https://github.com/hg096/gin_starter_kit.git

git fetch starter-kit
git merge starter-kit/main --allow-unrelated-histories
git commit -m "Merge starter-kit into project"
git push origin main
git fetch origin
