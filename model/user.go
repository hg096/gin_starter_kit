package model

import (
	"database/sql"
	"gin_starter/model/core"
	"gin_starter/util"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// string `db:"" validate:","`
// required 필수 / alphanum 알파벳과 숫자만 / min=6 최소 6 / max=32 최대 32 / alphaunicode 공백 없이 영문 또는 한글 / alpha 알파벳만 / email 이메일 / numeric 숫자 형식만 / gt / gte / lt / lte / len=10 정확한 길이 / regexp=^[a-zA-Z0-9]*$ 정규식으로 패턴 / eqfield=PasswordConfirm 다른 필드와 값이 동일한지

type UserInsert struct {
	core.BaseModel
	id               string `db:"u_id"`
	pass             string `db:"u_pass" validate:"min=6,max=20,required"`
	name             string `db:"u_name" validate:"max=30,required"`
	email            string `db:"u_email" validate:"email,required"`
	ValidateConverts map[string]string
	TableName        string
}
type UserUpdate struct {
	core.BaseModel
	id               string `db:"u_id"`
	pass             string `db:"u_pass" validate:"omitempty,min=6,max=20"`
	name             string `db:"u_name" validate:"omitempty,max=30"`
	email            string `db:"u_email" validate:"omitempty,email"`
	ValidateConverts map[string]string
	TableName        string
}

func NewUser() *UserInsert {
	return &UserInsert{
		BaseModel: core.NewBaseModel(),
		ValidateConverts: map[string]string{
			"id":    "아이디",
			"pass":  "비밀번호",
			"name":  "이름",
			"email": "이메일",
		},
		TableName: "_user",
	}
}
func NewUpUser() *UserUpdate {
	return &UserUpdate{
		BaseModel: core.NewBaseModel(),
		ValidateConverts: map[string]string{
			"id":    "아이디",
			"pass":  "비밀번호",
			"name":  "이름",
			"email": "이메일",
		},
		TableName: "_user",
	}
}

func (u *UserInsert) Insert(c *gin.Context, tx *sql.Tx,
	data map[string]string, errWhere string) (int64, error, error) {

	fieldMap := map[string]*string{
		"u_id":    &u.id,
		"u_pass":  &u.pass,
		"u_name":  &u.name,
		"u_email": &u.email,
	}
	util.AssignStringFields(data, fieldMap)

	err := core.ValidateModel(u)
	if core.HandleValidationError(c, tx, err, u.ValidateConverts) {
		return 0, err, nil
	}

	pass := data["u_pass"]
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "요청에 실패했습니다.", "errCode": 0})
		return 0, err, nil
	}
	data["u_pass"] = string(hashedPass)

	insertedID, err := core.BuildInsertQuery(c, tx, u.TableName, data, errWhere)
	if err != nil {
		return 0, nil, err
	}
	// log.Printf("User Insert 성공. Inserted ID: %d", insertedID)

	return insertedID, nil, nil
}

func (u *UserUpdate) Update(c *gin.Context, tx *sql.Tx,
	data map[string]string, where string, whereData []string, errWhere string) (error, error) {

	fieldMap := map[string]*string{
		"u_id":    &u.id,
		"u_pass":  &u.pass,
		"u_name":  &u.name,
		"u_email": &u.email,
	}
	util.AssignStringFields(data, fieldMap)

	err := core.ValidateModel(u)
	if core.HandleValidationError(c, tx, err, u.ValidateConverts) {
		return err, nil
	}

	if !util.EmptyString(data["u_pass"]) {
		pass := data["u_pass"]
		hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			return err, nil
		}
		data["u_pass"] = string(hashedPass)
	}
	delete(data, "u_id")

	_, err = core.BuildUpdateQuery(c, tx, u.TableName, data, where, whereData, errWhere)
	if err != nil {
		return nil, err
	}

	// 콘솔찍기위함
	// log.Printf("User update 성공. update ID: %s / %s", where, whereData)

	return nil, nil
}
