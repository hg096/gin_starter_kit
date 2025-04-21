package core

import (
	"sync"

	// "github.com/go-playground/validator/v10"
	v10 "github.com/go-playground/validator/v10"
)

// 모든 모델이 임베딩할 수 있는 베이스 구조체
type BaseModel struct {
	validate *v10.Validate
}

var (
	baseValidator *v10.Validate
	once          sync.Once
)

// 싱글톤 형태의 validator 인스턴스를 생성 및 반환
func getValidator() *v10.Validate {
	once.Do(func() {
		baseValidator = v10.New()
	})
	return baseValidator
}

// BaseModel의 인스턴스를 생성하는 생성자 함수
// 이 함수는 싱글톤 validator 인스턴스를 설정하여 반환
func NewBaseModel() BaseModel {
	return BaseModel{
		validate: getValidator(),
	}
}

// 전달받은 구조체에 대해 유효성 검사를 수행
func (bm *BaseModel) ValidateStruct(s interface{}) error {
	return bm.validate.Struct(s)
}
