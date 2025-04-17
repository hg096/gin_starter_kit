package model

import (
	"database/sql"
	"errors"
	"gin_starter/model/core"
	"gin_starter/util"
	"net/http"

	"github.com/gin-gonic/gin"
	v10 "github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

// string `db:"" validate:","`
// required 필수 / alphanum 알파벳과 숫자만 / min=6 최소 6 / max=32 최대 32 / alphaunicode 공백 없이 영문 또는 한글 / alpha 알파벳만 / email 이메일 / numeric 숫자 형식만 / gt / gte / lt / lte / len=10 정확한 길이 / regexp=^[a-zA-Z0-9]*$ 정규식으로 패턴 / eqfield=PasswordConfirm 다른 필드와 값이 동일한지

type UserInsert struct {
	core.BaseModel
	U_id    string `db:"u_id"`
	U_pass  string `db:"u_pass" validate:"min=6,max=20,required"`
	U_name  string `db:"u_name" validate:"max=30,required"`
	U_email string `db:"u_email" validate:"email,required"`
}
type UserUpdate struct {
	core.BaseModel
	U_id    string `db:"u_id"`
	U_pass  string `db:"u_pass" validate:"omitempty,min=6,max=20"`
	U_name  string `db:"u_name" validate:"omitempty,max=30"`
	U_email string `db:"u_email" validate:"omitempty,email"`
}

var ve v10.ValidationErrors

var validateConverts = map[string]string{
	"U_id":    "아이디",
	"U_pass":  "비밀번호",
	"U_name":  "이름",
	"U_email": "이메일",
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
		if errors.As(err, &ve) {
			msgs := core.FormatValidationErrors(ve, validateConverts)
			c.JSON(http.StatusBadRequest, gin.H{"errors": msgs})
		}
		return 0, err, nil
	}

	pass := data["u_pass"]
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "요청에 실패했습니다.", "errCode": 0})
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
	data map[string]string, where string, whereData []string, errWhere string) (error, error) {

	fieldMap := map[string]*string{
		// "u_id":    &u.U_id,
		"u_pass":  &u.U_pass,
		"u_name":  &u.U_name,
		"u_email": &u.U_email,
	}
	util.AssignStringFields(data, fieldMap)
	if err := core.ValidateModel(u); err != nil {
		if errors.As(err, &ve) {
			msgs := core.FormatValidationErrors(ve, validateConverts)
			c.JSON(http.StatusBadRequest, gin.H{"errors": msgs})
		}
		return err, nil
	}

	if !util.Empty(data["u_pass"]) {
		pass := data["u_pass"]
		hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			return err, nil
		}
		data["u_pass"] = string(hashedPass)
	}
	delete(data, "u_id")
	_, err := core.BuildUpdateQuery(c, tx, tableName, data, where, whereData, errWhere)
	if err != nil {
		return nil, err
	}

	// 콘솔찍기위함
	// log.Printf("User update 성공. update ID: %s / %s", where, whereData)

	return nil, nil
}
