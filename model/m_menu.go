package model

import (
	"database/sql"
	"gin_starter/model/dbCore"

	"github.com/gin-gonic/gin"
)

type MenuGroup struct {
	dbCore.BaseModel
	ValidateConverts map[string]string
	TableName        string
	Key              string     `db:"key"`
	Label            string     `db:"label"`
	Order            string     `db:"order"`
	Items            []MenuItem `db:"-"`
}

func NewMenuGroup() *MenuGroup {
	return &MenuGroup{
		BaseModel: dbCore.NewBaseModel(),
		// ValidateConverts: map[string]string{
		// 	"Key":   "구분명",
		// 	"Label": "그룹명",
		// 	"Order": "순서",
		// },
		TableName: "_menu_groups",
	}
}

type MenuItem struct {
	dbCore.BaseModel
	ValidateConverts map[string]string
	TableName        string
	Label            string   `db:"label"`
	Href             string   `db:"href"`
	Roles            []string `db:"roles"` // JSON으로 처리
	Order            string   `db:"order"`
}

func NewMenuItem() *MenuItem {
	return &MenuItem{
		BaseModel: dbCore.NewBaseModel(),
		// ValidateConverts: map[string]string{
		// 	"Label": "메뉴명",
		// 	"Href":  "주소",
		// 	"Roles": "권한",
		// 	"Order": "순서",
		// },
		TableName: "_menu_items",
	}
}

func (u *MenuGroup) AddMenuGroup(c *gin.Context, tx *sql.Tx,
	data map[string]string, errWhere string) (string, error, error) {

	insertedID, err := dbCore.BuildInsertQuery(c, tx, u.TableName, data, errWhere)
	if err != nil {
		return "0", nil, err
	}
	return insertedID, nil, nil
}

func (u *MenuGroup) UpdateMenuGroup(c *gin.Context, tx *sql.Tx,
	data map[string]string, where string, whereData []string, errWhere string) (error, error) {

	_, err := dbCore.BuildUpdateQuery(c, tx, u.TableName, data, where, whereData, errWhere)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (u *MenuGroup) DeleteMenuGroup(c *gin.Context, tx *sql.Tx,
	data map[string]string, where string, whereData []string, errWhere string) (error, error) {

	_, err := dbCore.BuildDeleteQuery(c, tx, u.TableName, where, whereData, errWhere)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (u *MenuItem) AddMenuItem(c *gin.Context, tx *sql.Tx,
	data map[string]string, errWhere string) (string, error, error) {

	insertedID, err := dbCore.BuildInsertQuery(c, tx, u.TableName, data, errWhere)
	if err != nil {
		return "0", nil, err
	}
	return insertedID, nil, nil
}

func (u *MenuItem) UpdateMenuItem(c *gin.Context, tx *sql.Tx,
	data map[string]string, where string, whereData []string, errWhere string) (error, error) {

	_, err := dbCore.BuildUpdateQuery(c, tx, u.TableName, data, where, whereData, errWhere)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (u *MenuItem) DeleteMenuItem(c *gin.Context, tx *sql.Tx,
	data map[string]string, where string, whereData []string, errWhere string) (error, error) {

	_, err := dbCore.BuildDeleteQuery(c, tx, u.TableName, where, whereData, errWhere)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
