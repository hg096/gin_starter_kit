package model

import (
	"database/sql"
	"gin_starter/model/core"
	"gin_starter/util"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
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

func (u *UserInsert) Insert(c *gin.Context, tx *sql.Tx,
	data map[string]string, errWhere string) (int64, error, error) {

	fieldMap := map[string]*string{
		"u_id":    &u.U_id,
		"u_pass":  &u.U_pass,
		"u_name":  &u.U_name,
		"u_email": &u.U_email,
	}
	util.AssignStringFields(data, fieldMap)
	if err := core.ValidateModel(u); err != nil {
		// !!!!!
		if vErrs, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range vErrs {
				// fieldErr.Field() : 오류가 발생한 필드 이름
				// fieldErr.Tag()   : 실패한 검증 태그 (예: "required", "email" 등)
				// fieldErr.Param() : 태그에 전달된 추가 파라미터 (예: min=6에서 "6")
				log.Printf("Field '%s' failed validation: %s (param: %s)",
					fieldErr.Field(), fieldErr.Tag(), fieldErr.Param())
			}
		}
		return 0, err, nil
	}

	pass := data["u_pass"]
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return 0, err, nil
	}
	data["u_pass"] = string(hashedPass)

	insertedID, err := core.BuildInsertQuery(c, tx, tableName, data, errWhere)
	if err != nil {
		return 0, nil, err
	}
	// log.Printf("User Insert 성공. Inserted ID: %d", insertedID)

	return insertedID, nil, nil
}

func (u *UserUpdate) Update(c *gin.Context, tx *sql.Tx,
	data map[string]string, where string, whereData []string, errWhere string) (sql.Result, error, error) {

	fieldMap := map[string]*string{
		"u_id":    &u.U_id,
		"u_pass":  &u.U_pass,
		"u_name":  &u.U_name,
		"u_email": &u.U_email,
	}
	util.AssignStringFields(data, fieldMap)
	if err := core.ValidateModel(u); err != nil {
		return nil, err, nil
	}

	if !util.Empty(data["u_pass"]) {
		pass := data["u_pass"]
		hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			return nil, err, nil
		}
		data["u_pass"] = string(hashedPass)
	}
	sqlResult, err := core.BuildUpdateQuery(c, tx, tableName, data, where, whereData, errWhere)
	if err != nil {
		return nil, nil, err
	}

	// 콘솔찍기위함
	// log.Printf("User update 성공. update ID: %s / %s", where, whereData)

	return sqlResult, nil, nil
}
