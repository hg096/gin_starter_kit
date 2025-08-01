package adm

import (
	"fmt"
	"gin_starter/model"
	"gin_starter/model/dbCore"
	"gin_starter/util/utilCore"
	"gin_starter/util/utilCore/auth"
	"gin_starter/util/utilCore/pageUtil"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func SetupAdminApiRoutes(rg *gin.RouterGroup) {

	adminApiGroup := rg.Group("/api")
	{

		adminApiGroup.POST("/make", func(c *gin.Context) { apiUserMake(c) })

		adminApiGroup.POST("/makeUp", func(c *gin.Context) { apiUserMakeUp(c) })

		adminApiGroup.POST("/logIn", func(c *gin.Context) { apiUserLogIn(c) })

		adminApiGroup.GET("/menus", func(c *gin.Context) { apiAdmMenus(c) })

		// 메뉴 그룹 수정
		adminApiGroup.PUT("/menus/group/:id", func(c *gin.Context) { apiAdmMenusGroupEdit(c) })
		// 메뉴 그룹 삭제
		adminApiGroup.DELETE("/menus/group/:id", func(c *gin.Context) { apiAdmMenusGroupDel(c) })

		// 메뉴 추가
		adminApiGroup.POST("/menus/item", func(c *gin.Context) { apiAdmMenusItemAdd(c) })
		// 메뉴 수정
		adminApiGroup.PUT("/menus/item/:id", func(c *gin.Context) { apiAdmMenusItemEdit(c) })
		// 메뉴 삭제
		adminApiGroup.DELETE("/menus/item/:id", func(c *gin.Context) { apiAdmMenusItemDel(c) })

		// 사용자 목록
		adminApiGroup.GET("/users", func(c *gin.Context) { apiAdmUserList(c) })
		// 사용자 추가
		adminApiGroup.POST("/users", func(c *gin.Context) { apiAdmUserAdd(c) })
		// 사용자 수정
		adminApiGroup.PUT("/users/:id", func(c *gin.Context) { apiAdmUserEdit(c) })
		// 사용자 삭제
		adminApiGroup.DELETE("/users/:id", func(c *gin.Context) { apiAdmUserDel(c) })

	}
}

// 가입
func apiUserMake(c *gin.Context) {

	user := model.NewUser()

	data := utilCore.PostFields(c, map[string][2]string{
		"user_id":    {"u_id", ""},
		"user_pass":  {"u_pass", ""},
		"user_name":  {"u_name", ""},
		"user_email": {"u_email", ""},
	})

	data["u_auth_type"] = "AG"

	// 트랜젝션 예시 불필요할시 제거
	tx, err := dbCore.BeginTransaction(c)
	if err != nil {
		return
	}

	insertedID, valErr, sqlErr := user.Insert(c, tx, data, "api/user/make")
	if valErr != nil || sqlErr != nil {
		log.Printf("User Insert 에러: %v", valErr)
		return
	}

	// 트랜젝션 예시 불필요할시 제거
	if cerr := dbCore.EndTransactionCommit(tx); cerr != nil {
		return
	}

	fmt.Printf("User가 성공적으로 추가 되었습니다. Inserted ID: %s\n", insertedID)

	utilCore.EndResponse(c, http.StatusOK, gin.H{"message": "User make"}, "rest /user/make")
}

// 수정
func apiUserMakeUp(c *gin.Context) {

	user := model.NewUpUser()

	data := utilCore.PostFields(c, map[string][2]string{
		"user_id": {"u_id", ""},
		// "user_pass":  {"u_pass", ""},
		// "user_name":  {"u_name", ""},
		"user_email": {"u_email", ""},
	})

	valErr, sqlErr := user.Update(c, nil, data, "u_id = ?", []string{data["u_id"]}, "api/user/makeUp")
	if valErr != nil || sqlErr != nil {
		log.Printf("User Insert 에러: %v", valErr)
		return
	} else {
		// fmt.Printf("User가 성공적으로 수정 되었습니다. Inserted ID: %s\n", sqlResult)
	}

	utilCore.EndResponse(c, http.StatusOK, gin.H{"message": "User update"}, "rest /user/makeUp")
}

// 로그인
func apiUserLogIn(c *gin.Context) {

	data := utilCore.PostFields(c, map[string][2]string{
		"user_id":   {"u_id", ""},
		"user_pass": {"u_pass", ""},
	})

	userRows, err := dbCore.BuildSelectQuery(c, nil,
		"SELECT u_pass FROM _user WHERE u_id = ? LIMIT 1", []string{data["u_id"]}, "apiUserLogIn-getPass")

	if err != nil || len(userRows) == 0 {
		utilCore.EndResponse(c, http.StatusUnauthorized, gin.H{}, "rest /user/login getUser")
		return
	}
	storedHash := userRows[0]["u_pass"]

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(data["u_pass"]))
	if err != nil {
		utilCore.EndResponse(c, http.StatusUnauthorized, gin.H{}, "rest /user/login pass")
		return
	}

	at, rt, err := auth.GenerateTokens(data["u_id"], "")
	if err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{}, "rest /user/login-GenerateTokens")
		return
	}

	_, err = dbCore.BuildUpdateQuery(c, nil, "_user", map[string]string{"u_re_token": rt}, "u_id = ?", []string{data["u_id"]}, "fn apiUserLogIn-BuildUpdateQuery")
	if err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{}, "fn apiUserLogIn-BuildUpdateQuery")
		return
	}

	pageUtil.SetCookie(c, "acc_token", at, 60*15)
	pageUtil.SetCookie(c, "ref_token", rt, 60*60*24*7)

	// c.Redirect(http.StatusFound, "/adm")
	utilCore.EndResponse(c, http.StatusOK, gin.H{}, "rest /user/login")
}

// 메뉴 리스트
func apiAdmMenus(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "", 0)

	userType, _ := utilCore.GetContextVal(c, "user_type")
	menuData := pageUtil.MakeMenuRole(c, userType, true)

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": menuData}, "rest /user/login")
}

// 메뉴 그룹 수정
func apiAdmMenusGroupEdit(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "A", 0)

	getData := utilCore.GetFields(c, map[string][2]string{
		"id": {"id", "0"},
	})

	postData := utilCore.PostFields(c, map[string][2]string{
		"Label": {"mg_label", ""},
		"Order": {"mg_order", "1"},
	})

	dbCore.BuildUpdateQuery(c, nil, "_menu_groups", postData, "mg_idx = ?", []string{getData["id"]}, "apiAdmMenusGroupEdit")

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": ""}, "rest apiAdmMenusGroupEdit")
}

// 메뉴 그룹 삭제
func apiAdmMenusGroupDel(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "A", 0)

	getData := utilCore.GetFields(c, map[string][2]string{
		"id": {"id", "0"},
	})

	dbCore.BuildDeleteQuery(c, nil, "_menu_groups", "mg_idx = ?", []string{getData["id"]}, "apiAdmMenusGroupDel")
	dbCore.BuildDeleteQuery(c, nil, "_menu_items", "mi_group_id = ?", []string{getData["id"]}, "apiAdmMenusGroupDel")

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": ""}, "rest apiAdmMenusGroupDel")
}

// 메뉴 추가
func apiAdmMenusItemAdd(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "A", 0)

	postData := utilCore.PostFields(c, map[string][2]string{
		"Label":    {"mi_label", ""},
		"Order":    {"mi_order", "1"},
		"Href":     {"mi_href", ""},
		"Role":     {"mi_roles", ""},
		"group_id": {"mi_group_id", "0"},
	})

	tx, err := dbCore.BeginTransaction(c)
	if err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{"data": ""}, "rest apiAdmMenusItemAdd")
		return
	}

	if postData["mi_group_id"] == "0" {
		postData2 := utilCore.PostFields(c, map[string][2]string{
			"LabelG": {"mg_label", ""},
			"OrderG": {"mg_order", "1"},
		})
		insertKey, _ := dbCore.BuildInsertQuery(c, tx, "_menu_groups", postData2, "apiAdmMenusItemAdd")
		postData["mi_group_id"] = insertKey
	}

	dbCore.BuildInsertQuery(c, tx, "_menu_items", postData, "apiAdmMenusItemAdd")

	if err := dbCore.EndTransactionCommit(tx); err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{"data": ""}, "rest apiAdmMenusItemAdd")
		return
	}

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": ""}, "rest apiAdmMenusItemAdd")
}

// 메뉴 수정
func apiAdmMenusItemEdit(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "A", 0)

	getData := utilCore.GetFields(c, map[string][2]string{
		"id": {"id", "0"},
	})

	postData := utilCore.PostFields(c, map[string][2]string{
		"Label": {"mi_label", ""},
		"Order": {"mi_order", "1"},
		"Href":  {"mi_href", ""},
		"Role":  {"mi_roles", ""},
	})

	dbCore.BuildUpdateQuery(c, nil, "_menu_items", postData, "mi_idx = ?", []string{getData["id"]}, "apiAdmMenusItemEdit")

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": ""}, "rest apiAdmMenusItemEdit")
}

// 메뉴 삭제
func apiAdmMenusItemDel(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "A", 0)

	getData := utilCore.GetFields(c, map[string][2]string{
		"id": {"id", "0"},
	})

	dataMi, err := dbCore.BuildSelectQuery(c, nil, `
			SELECT count(mi2.mi_idx) AS CNT, mg_idx
			FROM _menu_items mi
			join _menu_groups mg on mi.mi_group_id = mg.mg_idx
			join _menu_items mi2 on mi2.mi_group_id = mg.mg_idx
			where mi.mi_idx = ?
			GROUP BY mg.mg_idx `, []string{getData["id"]}, "get Menu sql err")
	if err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{"data": ""}, "rest apiAdmMenusItemDel")
		return
	}
	menuCnt, _ := strconv.Atoi(dataMi[0]["CNT"])

	tx, err := dbCore.BeginTransaction(c)
	if err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{"data": ""}, "rest apiAdmMenusItemDel")
		return
	}

	if 2 > menuCnt {
		dbCore.BuildDeleteQuery(c, tx, "_menu_groups", "mg_idx = ?", []string{dataMi[0]["mg_idx"]}, "apiAdmMenusItemDel")
		dbCore.BuildDeleteQuery(c, tx, "_menu_items", "mi_group_id = ?", []string{dataMi[0]["mg_idx"]}, "apiAdmMenusItemDel2")
	}
	dbCore.BuildDeleteQuery(c, tx, "_menu_items", "mi_idx = ?", []string{getData["id"]}, "apiAdmMenusItemDel")

	if err := dbCore.EndTransactionCommit(tx); err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{"data": ""}, "rest apiAdmMenusItemDel")
		return
	}

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": ""}, "rest apiAdmMenusItemDel")
}

// 사용자 목록
func apiAdmUserList(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "A, M, AG", 0)

	users, err := dbCore.BuildSelectQuery(c, nil, `SELECT u_idx, u_id, u_name, u_email, u_auth_type FROM _user ORDER BY u_idx`, []string{}, "apiAdmUserList")
	if err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{"data": ""}, "rest apiAdmUserList")
		return
	}

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": users}, "rest apiAdmUserList")
}

// 사용자 추가
func apiAdmUserAdd(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "A", 0)

	user := model.NewUser()
	data := utilCore.PostFields(c, map[string][2]string{
		"user_id":    {"u_id", ""},
		"user_pass":  {"u_pass", ""},
		"user_name":  {"u_name", ""},
		"user_email": {"u_email", ""},
		"user_auth":  {"u_auth_type", "AG"},
	})

	tx, err := dbCore.BeginTransaction(c)
	if err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{"data": ""}, "rest apiAdmUserAdd")
		return
	}

	_, valErr, sqlErr := user.Insert(c, tx, data, "apiAdmUserAdd")
	if valErr != nil || sqlErr != nil {
		log.Printf("User Insert 에러: %v", valErr)
		return
	}

	if err := dbCore.EndTransactionCommit(tx); err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{"data": ""}, "rest apiAdmUserAdd")
		return
	}

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": ""}, "rest apiAdmUserAdd")
}

// 사용자 수정
func apiAdmUserEdit(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "A", 0)

	getData := utilCore.GetFields(c, map[string][2]string{
		"id": {"id", "0"},
	})
	postData := utilCore.PostFields(c, map[string][2]string{
		"user_name":  {"u_name", ""},
		"user_email": {"u_email", ""},
		"user_auth":  {"u_auth_type", ""},
		"user_pass":  {"u_pass", ""},
	})

	user := model.NewUpUser()
	valErr, sqlErr := user.Update(c, nil, postData, "u_idx = ?", []string{getData["id"]}, "apiAdmUserEdit")
	if valErr != nil || sqlErr != nil {
		log.Printf("User Update 에러: %v", valErr)
		return
	}

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": ""}, "rest apiAdmUserEdit")
}

// 사용자 삭제
func apiAdmUserDel(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "A", 0)

	getData := utilCore.GetFields(c, map[string][2]string{
		"id": {"id", "0"},
	})

	dbCore.BuildDeleteQuery(c, nil, "_user", "u_idx = ?", []string{getData["id"]}, "apiAdmUserDel")

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": ""}, "rest apiAdmUserDel")
}
