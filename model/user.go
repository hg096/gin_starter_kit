package model

import (
	"database/sql"
	"gin_starter/model/core"
	"gin_starter/util"
	"log"

	"golang.org/x/crypto/bcrypt"
)

// string `db:"" validate:","`
// required 필수 / alphanum 알파벳과 숫자만 / min=6 최소 6 / max=32 최대 32 / alphaunicode 공백 없이 영문 또는 한글 / alpha 알파벳만 / email 이메일 / numeric 숫자 형식만 / gt / gte / lt / lte / len=10 정확한 길이 / regexp=^[a-zA-Z0-9]*$ 정규식으로 패턴 / eqfield=PasswordConfirm 다른 필드와 값이 동일한지

type UserInsert struct {
	core.BaseModel
	U_id    string `db:"u_id"` // Duplicate entry
	U_pass  string `db:"u_pass" validate:"min=6,max=20,required"`
	U_name  string `db:"u_name" validate:"max=30,required"`
	U_email string `db:"u_email" validate:"email,required"`
}
type UserUpdate struct {
	core.BaseModel
	U_id    string `db:"u_id"` // Duplicate entry
	U_pass  string `db:"u_pass" validate:"min=6,max=20"`
	U_name  string `db:"u_name" validate:"max=30"`
	U_email string `db:"u_email" validate:"email"`
}

var tableName string = "_user"

func NewUser() *UserInsert {
	return &UserInsert{
		BaseModel: core.NewBaseModel(),
	}
}
func NewUpUser() *UserUpdate {
	return &UserUpdate{
		BaseModel: core.NewBaseModel(),
	}
}

func (u *UserInsert) Insert(data map[string]interface{}) (int64, error) {

	fieldMap := map[string]*string{
		"u_id":    &u.U_id,
		"u_pass":  &u.U_pass,
		"u_name":  &u.U_name,
		"u_email": &u.U_email,
	}
	util.AssignStringFields(data, fieldMap)
	if err := core.ValidateModel(u); err != nil {
		return 0, err
	}

	if pass, ok := data["u_pass"].(string); ok {
		hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			return 0, err
		}
		data["u_pass"] = string(hashedPass)
	}

	insertedID, err := core.BuildInsertQuery(tableName, data)
	if err != nil {
		return 0, err
	}
	log.Printf("User Insert 성공. Inserted ID: %d", insertedID)

	return insertedID, nil
}

func (u *UserUpdate) Update(data map[string]interface{}, where string, whereData []interface{}) (sql.Result, error) {

	fieldMap := map[string]*string{
		"u_id":    &u.U_id,
		"u_pass":  &u.U_pass,
		"u_name":  &u.U_name,
		"u_email": &u.U_email,
	}
	util.AssignStringFields(data, fieldMap)
	if err := core.ValidateModel(u); err != nil {
		return nil, err
	}

	if !util.Empty(data["u_pass"]) {
		if pass, ok := data["u_pass"].(string); ok {
			hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}
			data["u_pass"] = string(hashedPass)
		}
	}
	sqlResult, err := core.BuildUpdateQuery(tableName, data, where, whereData)
	if err != nil {
		return nil, err
	}

	// 콘솔찍기위함
	log.Printf("User update 성공. update ID: %s / %s", where, whereData)

	return sqlResult, nil
}
