package model

import (
	"gin_starter/model/core"
	"log"
)

// User 모델은 _user 테이블과 연계된다고 가정합니다.
type User struct {
	core.BaseModel        // BaseModel에 ValidateStruct 등 유효성 검사 기능 포함
	ID             int    `db:"id"`
	Name           string `db:"name" validate:"required"`
	Email          string `db:"email" validate:"required,email"`
	// 추가 필드가 있다면 여기에 정의합니다.
}

var tableName string = "_user"

// NewUser는 빈 User 객체를 생성합니다.
func NewUser() *User {
	return &User{
		BaseModel: core.NewBaseModel(),
	}
}

// Validate는 User 모델 전용 유효성 검사 메서드입니다.
func (u *User) Validate() error {
	return u.ValidateStruct(u)
}

// Insert는 추가할 컬럼명 => 값 형태의 데이터를 인자로 받아, 유효성 검사를 수행한 후 DB에 INSERT 쿼리를 실행합니다.
func (u *User) Insert(data map[string]interface{}) (int64, error) {
	// (선택사항) 여기서 data에 필요한 전처리나 모델별 유효성 검사를 진행할 수 있습니다.
	// 예를 들어, 필수 컬럼("name", "email")의 존재 여부를 체크하는 로직을 추가할 수 있습니다.
	if err := u.Validate(); err != nil { // BaseModel에 정의된 Validate()로 구조체 자체의 유효성 검사도 가능
		return 0, err
	}

	// _user 테이블에 데이터를 삽입하는 쿼리와 인자를 생성 및 실행합니다.
	insertedID, err := core.BuildInsertQuery(tableName, data)
	if err != nil {
		return 0, err
	}
	log.Printf("User Insert 성공. Inserted ID: %d", insertedID)

	return insertedID, nil
}
