package core

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

// BaseModel은 모든 모델이 임베딩할 수 있는 베이스 구조체입니다.
type BaseModel struct {
	validate *validator.Validate
}

var (
	baseValidator *validator.Validate
	once          sync.Once
)

// getValidator는 싱글톤 형태의 validator 인스턴스를 생성 및 반환합니다.
func getValidator() *validator.Validate {
	once.Do(func() {
		baseValidator = validator.New()
	})
	return baseValidator
}

// NewBaseModel은 BaseModel의 인스턴스를 생성하는 생성자 함수입니다.
// 이 함수는 싱글톤 validator 인스턴스를 설정하여 반환합니다.
func NewBaseModel() BaseModel {
	return BaseModel{
		validate: getValidator(),
	}
}

// ValidateStruct는 전달받은 구조체에 대해 유효성 검사를 수행합니다.
func (bm *BaseModel) ValidateStruct(s interface{}) error {
	return bm.validate.Struct(s)
}
