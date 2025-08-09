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

	rules := []utilCore.InputRule{
		{InputKey: "user_id", OutputKey: "u_id", Label: "아이디",
			Allow: utilCore.AllowKorEngNum, Required: true, MinLen: 3, MaxLen: 20, TrimSpace: true},
		{InputKey: "user_pass", OutputKey: "u_pass", Label: "비밀번호",
			Allow: utilCore.AllowKorEngNumSp, Required: true, MinLen: 1, MaxLen: 50, TrimSpace: true},
		{InputKey: "user_name", OutputKey: "u_name", Label: "이름",
			Allow: utilCore.AllowKorean, Required: true, MinLen: 2, MaxLen: 10, TrimSpace: true},
		{InputKey: "user_email", OutputKey: "u_email", Label: "이메일",
			Allow: utilCore.AllowEmailLike, Required: true, MinLen: 5, MaxLen: 100, TrimSpace: true},
	}

	data, ok := utilCore.BindAutoAndRespond(c, rules, "apiUserMake BindAutoAndRespond")
	if !ok {
		return
	}

	data["u_auth_type"] = "AG"

	user := model.NewUser()
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

	rules := []utilCore.InputRule{
		{InputKey: "user_id", OutputKey: "u_id", Label: "아이디",
			Allow: utilCore.AllowKorEngNum, Required: true, MinLen: 3, MaxLen: 20, TrimSpace: true},
		// {InputKey: "user_pass", OutputKey: "u_pass", Label: "비밀번호",
		// 	Allow: utilCore.AllowKorEngNumSp, Required: true, MinLen: 1, MaxLen: 50, TrimSpace: true},
		// {InputKey: "user_name", OutputKey: "u_name", Label: "이름",
		// 	Allow: utilCore.AllowKorean, Required: true, MinLen: 2, MaxLen: 10, TrimSpace: true},
		{InputKey: "user_email", OutputKey: "u_email", Label: "이메일",
			Allow: utilCore.AllowEmailLike, Required: true, MinLen: 5, MaxLen: 100, TrimSpace: true},
	}

	data, ok := utilCore.BindAutoAndRespond(c, rules, "apiUserMakeUp BindAutoAndRespond")
	if !ok {
		return
	}

	user := model.NewUpUser()
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

	rules := []utilCore.InputRule{
		{InputKey: "user_id", OutputKey: "u_id", Label: "아이디",
			Allow: utilCore.AllowKorEngNum, Required: true, MinLen: 3, MaxLen: 20, TrimSpace: true},
		{InputKey: "user_pass", OutputKey: "u_pass", Label: "비밀번호",
			Allow: utilCore.AllowKorEngNumSp, Required: true, MinLen: 1, MaxLen: 50, TrimSpace: true},
	}

	data, ok := utilCore.BindAutoAndRespond(c, rules, "apiUserLogIn BindAutoAndRespond")
	if !ok {
		return
	}

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

	rulesGet := []utilCore.InputRule{
		{InputKey: "id", OutputKey: "id", Label: "ID",
			Allow: utilCore.AllowNumber, Required: true, Default: "0", TrimSpace: true},
	}
	getData, ok := utilCore.BindAutoAndRespond(c, rulesGet, "apiAdmMenusGroupEdit BindAutoAndRespond")
	if !ok {
		return
	}

	rulesPost := []utilCore.InputRule{
		{InputKey: "Label", OutputKey: "mg_label", Label: "이름",
			Allow: utilCore.AllowKorEngNum, Required: true, MinLen: 1, MaxLen: 20, TrimSpace: true},
		{InputKey: "Order", OutputKey: "mg_order", Label: "정렬",
			Allow: utilCore.AllowNumber, Required: true, Min: 1, Max: 99, TrimSpace: true},
	}
	postData, ok := utilCore.BindAutoAndRespond(c, rulesPost, "apiAdmMenusGroupEdit BindAutoAndRespond")
	if !ok {
		return
	}

	dbCore.BuildUpdateQuery(c, nil, "_menu_groups", postData, "mg_idx = ?", []string{getData["id"]}, "apiAdmMenusGroupEdit")

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": ""}, "rest apiAdmMenusGroupEdit")
}

// 메뉴 그룹 삭제
func apiAdmMenusGroupDel(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "A", 0)

	rulesGet := []utilCore.InputRule{
		{InputKey: "id", OutputKey: "id", Label: "ID",
			Allow: utilCore.AllowNumber, Required: true, Default: "0", TrimSpace: true},
	}
	getData, ok := utilCore.BindAutoAndRespond(c, rulesGet, "apiAdmMenusGroupDel BindAutoAndRespond")
	if !ok {
		return
	}

	dbCore.BuildDeleteQuery(c, nil, "_menu_groups", "mg_idx = ?", []string{getData["id"]}, "apiAdmMenusGroupDel")
	dbCore.BuildDeleteQuery(c, nil, "_menu_items", "mi_group_id = ?", []string{getData["id"]}, "apiAdmMenusGroupDel")

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": ""}, "rest apiAdmMenusGroupDel")
}

// 메뉴 추가
func apiAdmMenusItemAdd(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "A", 0)

	rulesPost := []utilCore.InputRule{
		{InputKey: "Label", OutputKey: "mi_label", Label: "이름",
			Allow: utilCore.AllowKorEngNum, Required: true, MinLen: 1, MaxLen: 20, TrimSpace: true},
		{InputKey: "Order", OutputKey: "mi_order", Label: "정렬",
			Allow: utilCore.AllowNumber, Required: true, Min: 1, Max: 99, TrimSpace: true},
		{InputKey: "Href", OutputKey: "mi_href", Label: "주소",
			Allow: utilCore.AllowKorEngNum, Required: true, TrimSpace: true},
		{InputKey: "Role", OutputKey: "mi_roles", Label: "권한",
			Allow: utilCore.AllowSafeText, Required: true, TrimSpace: true},
		{InputKey: "group_id", OutputKey: "mi_group_id", Label: "그룹 ID",
			Allow: utilCore.AllowNumber, Required: true, Min: 1, TrimSpace: true},
	}
	postData, ok := utilCore.BindAutoAndRespond(c, rulesPost, "apiAdmMenusItemAdd BindAutoAndRespond")
	if !ok {
		return
	}

	tx, err := dbCore.BeginTransaction(c)
	if err != nil {
		utilCore.EndResponse(c, http.StatusBadRequest, gin.H{"data": ""}, "rest apiAdmMenusItemAdd")
		return
	}

	if postData["mi_group_id"] == "0" {
		rulesPost := []utilCore.InputRule{
			{InputKey: "LabelG", OutputKey: "mg_label", Label: "이름",
				Allow: utilCore.AllowKorEngNum, Required: true, MinLen: 1, MaxLen: 20, TrimSpace: true},
			{InputKey: "OrderG", OutputKey: "mg_order", Label: "정렬",
				Allow: utilCore.AllowNumber, Required: true, Min: 1, Max: 99, TrimSpace: true},
		}
		postData2, ok := utilCore.BindAutoAndRespond(c, rulesPost, "apiAdmMenusItemAdd BindAutoAndRespond")
		if !ok {
			return
		}

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

	rulesGet := []utilCore.InputRule{
		{InputKey: "id", OutputKey: "id", Label: "ID",
			Allow: utilCore.AllowNumber, Required: true, Default: "0", TrimSpace: true},
	}
	getData, ok := utilCore.BindAutoAndRespond(c, rulesGet, "apiAdmMenusItemEdit BindAutoAndRespond")
	if !ok {
		return
	}

	rulesPost := []utilCore.InputRule{
		{InputKey: "Label", OutputKey: "mi_label", Label: "이름",
			Allow: utilCore.AllowKorEngNum, Required: true, MinLen: 1, MaxLen: 20, TrimSpace: true},
		{InputKey: "Order", OutputKey: "mi_order", Label: "정렬",
			Allow: utilCore.AllowNumber, Required: true, Min: 1, Max: 99, TrimSpace: true},
		{InputKey: "Href", OutputKey: "mi_href", Label: "주소",
			Allow: utilCore.AllowKorEngNum, Required: true, TrimSpace: true},
		{InputKey: "Role", OutputKey: "mi_roles", Label: "권한",
			Allow: utilCore.AllowSafeText, Required: true, TrimSpace: true},
	}
	postData, ok := utilCore.BindAutoAndRespond(c, rulesPost, "apiAdmMenusItemEdit BindAutoAndRespond")
	if !ok {
		return
	}

	dbCore.BuildUpdateQuery(c, nil, "_menu_items", postData, "mi_idx = ?", []string{getData["id"]}, "apiAdmMenusItemEdit")

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": ""}, "rest apiAdmMenusItemEdit")
}

// 메뉴 삭제
func apiAdmMenusItemDel(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "A", 0)

	rulesGet := []utilCore.InputRule{
		{InputKey: "id", OutputKey: "id", Label: "ID",
			Allow: utilCore.AllowNumber, Required: true, Default: "0", TrimSpace: true},
	}
	getData, ok := utilCore.BindAutoAndRespond(c, rulesGet, "apiAdmMenusItemDel BindAutoAndRespond")
	if !ok {
		return
	}

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

	rules := []utilCore.InputRule{
		{InputKey: "user_id", OutputKey: "u_id", Label: "아이디",
			Allow: utilCore.AllowKorEngNum, Required: true, MinLen: 3, MaxLen: 20, TrimSpace: true},
		{InputKey: "user_pass", OutputKey: "u_pass", Label: "비밀번호",
			Allow: utilCore.AllowKorEngNumSp, Required: true, MinLen: 1, MaxLen: 50, TrimSpace: true},
		{InputKey: "user_name", OutputKey: "u_name", Label: "이름",
			Allow: utilCore.AllowKorean, Required: true, MinLen: 2, MaxLen: 10, TrimSpace: true},
		{InputKey: "user_email", OutputKey: "u_email", Label: "이메일",
			Allow: utilCore.AllowEmailLike, Required: true, MinLen: 5, MaxLen: 100, TrimSpace: true},
		{InputKey: "user_auth", OutputKey: "u_auth_type", Label: "권한",
			Allow: utilCore.AllowEnglish, Required: true, Default: "AG", TrimSpace: true},
	}

	data, ok := utilCore.BindAutoAndRespond(c, rules, "apiAdmUserAdd BindAutoAndRespond")
	if !ok {
		return
	}

	user := model.NewUser()
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

	rulesGet := []utilCore.InputRule{
		{InputKey: "id", OutputKey: "id", Label: "ID",
			Allow: utilCore.AllowNumber, Required: true, Default: "0", TrimSpace: true},
	}
	getData, ok := utilCore.BindAutoAndRespond(c, rulesGet, "apiAdmUserEdit BindAutoAndRespond")
	if !ok {
		return
	}

	rules := []utilCore.InputRule{
		// {InputKey: "user_id", OutputKey: "u_id", Label: "아이디",
		// 	Allow: utilCore.AllowKorEngNum, Required: true, MinLen: 3, MaxLen: 20, TrimSpace: true},
		{InputKey: "user_pass", OutputKey: "u_pass", Label: "비밀번호",
			Allow: utilCore.AllowKorEngNumSp, Required: true, MinLen: 1, MaxLen: 50, TrimSpace: true},
		{InputKey: "user_name", OutputKey: "u_name", Label: "이름",
			Allow: utilCore.AllowKorean, Required: true, MinLen: 2, MaxLen: 10, TrimSpace: true},
		{InputKey: "user_email", OutputKey: "u_email", Label: "이메일",
			Allow: utilCore.AllowEmailLike, Required: true, MinLen: 5, MaxLen: 100, TrimSpace: true},
		{InputKey: "user_auth", OutputKey: "u_auth_type", Label: "권한",
			Allow: utilCore.AllowEnglish, Required: true, Default: "AG", TrimSpace: true},
	}

	postData, ok := utilCore.BindAutoAndRespond(c, rules, "apiAdmUserEdit BindAutoAndRespond")
	if !ok {
		return
	}

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

	rulesGet := []utilCore.InputRule{
		{InputKey: "id", OutputKey: "id", Label: "ID",
			Allow: utilCore.AllowNumber, Required: true, Default: "0", TrimSpace: true},
	}
	getData, ok := utilCore.BindAutoAndRespond(c, rulesGet, "apiAdmUserDel BindAutoAndRespond")
	if !ok {
		return
	}

	dbCore.BuildDeleteQuery(c, nil, "_user", "u_idx = ?", []string{getData["id"]}, "apiAdmUserDel")

	utilCore.EndResponse(c, http.StatusOK, gin.H{"data": ""}, "rest apiAdmUserDel")
}
