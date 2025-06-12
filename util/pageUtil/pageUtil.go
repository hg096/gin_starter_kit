package pageUtil

import (
	"encoding/json"
	"fmt"
	"gin_starter/model/core"
	"gin_starter/util"
	"gin_starter/util/auth"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"
)

type MenuItem struct {
	Label string
	Href  string
	Roles []string // 접근 가능한 권한
}

type MenuGroup struct {
	Key   string
	Label string
	Items []MenuItem
}

// 페이지 출력
func RenderPage(c *gin.Context, page string, customData gin.H, isMakeMenu bool) {

	data := gin.H{
		// "IsLoggedIn": true,
		"UserName":   "",
		"ShowFooter": true,
		"Menus":      []map[string]interface{}{},
	}

	for k, v := range customData {
		data[k] = v
	}

	tmpl, err := template.ParseFiles(
		"templates/adm/layouts/layout.tmpl",
		fmt.Sprintf("templates/adm/pages/%s.tmpl", page),
		"templates/adm/components/navbar.tmpl",
		"templates/adm/components/sidebar.tmpl",
		"templates/adm/components/footer.tmpl",
	)
	if err != nil {
		log.Fatalf("[종료] 템플릿 로딩 실패: %v", err)
	}

	if isMakeMenu {
		userType, _ := util.GetContextVal(c, "user_type")
		data["Menus"] = MakeMenuRole(c, userType, false)
	}

	// log.Println("RenderPage ")
	// log.Println(data)

	c.Status(http.StatusOK)
	tmpl.ExecuteTemplate(c.Writer, "layout", data)
}

// 로그인 체크, 토큰 체크, 만료시 쿠키갱신, 유저 접근 권한 체크
func RenderPageCheckLogin(c *gin.Context, userType string, lv int8) []map[string]interface{} {

	token, _ := c.Cookie("acc_token")
	// if err != nil || token == "" {
	// 	c.Redirect(http.StatusFound, "/adm/manage/login")
	// 	c.Abort()
	// 	return nil
	// }

	refToken, err := c.Cookie("ref_token")
	if err != nil || refToken == "" {
		c.Redirect(http.StatusFound, "/adm/manage/login")
		c.Abort()
		return nil
	}

	claims, err := auth.ValidateToken(token, auth.AccessSecret, auth.TokenSecret)
	if err != nil {
		newAT, newRT, errMsg := auth.RefreshHandler(c, map[string]string{"refresh_token": refToken})
		if !util.EmptyString(errMsg) {
			util.EndResponse(c, http.StatusBadRequest, gin.H{}, errMsg)
			return nil
		}
		claims, _ = auth.ValidateToken(newAT, auth.AccessSecret, auth.TokenSecret)
		SetCookie(c, "acc_token", newAT, 60*15)
		SetCookie(c, "ref_token", newRT, 60*60*24*7)
	}

	result, err := core.BuildSelectQuery(c, nil, "select u_auth_type, u_auth_level from _user where u_id = ? AND u_auth_type != 'U' ", []string{claims.JWTUserID}, "JWTAuthMiddleware.err")
	if err != nil {
		c.Redirect(http.StatusFound, "/adm/manage/login")
		c.Abort()
		return nil
	}

	// 사용자 타입 찾기
	if !util.EmptyString(userType) && result[0]["u_auth_type"] != userType {
		// 만약에 타입이 두가지 이상 들어가야할때
		index := strings.Index(userType, result[0]["u_auth_type"])
		if index < 0 {
			c.Redirect(http.StatusFound, "/adm/manage/login")
			c.Abort()
			return nil
		}
		return nil
	}

	// 등급 레벨 조건이 맞는지 확인
	if lv > 0 {
		u_auth_level, _ := util.StringToNumeric[int8](result[0]["u_auth_level"])
		if lv > u_auth_level {
			c.Redirect(http.StatusFound, "/adm/manage/login")
			c.Abort()
			return nil
		}
	}
	c.Set("user_id", claims.JWTUserID)
	c.Set("user_type", result[0]["u_auth_type"])
	c.Set("user_level", result[0]["u_auth_level"])

	return nil
}

// 메뉴 생성
func MakeMenuRole(c *gin.Context, userRole string, isOutRole bool) []map[string]interface{} {
	groupMap := map[string]map[string]interface{}{}
	orderMap := map[string]bool{}
	orderList := []string{}

	dataMi, err := core.BuildSelectQuery(c, nil, `
			SELECT
				mg.mg_idx AS group_id,
				mg.mg_label AS group_label,
				mg.mg_order AS group_order,
				mi.mi_group_id AS item_group,
				mi.mi_idx AS item_id,
				mi.mi_label AS item_label,
				mi.mi_href AS item_href,
				mi.mi_roles AS item_roles,
				mi.mi_order AS item_order
			FROM _menu_items mi
			left join _menu_groups mg on mi.mi_group_id = mg.mg_idx
			ORDER BY mg.mg_order, mi.mi_order`, []string{}, "get Menu sql err")
	if err != nil {
		c.Redirect(http.StatusFound, "/adm/manage/login")
		c.Abort()
		return nil
	}

	for _, row := range dataMi {
		roles := []string{}
		if err := json.Unmarshal([]byte(row["item_roles"]), &roles); err != nil || !contains(roles, userRole) {
			continue
		}

		keyGroup := row["group_id"]
		if _, exists := groupMap[keyGroup]; !exists {
			groupMap[keyGroup] = map[string]interface{}{
				"ID":      keyGroup,
				"Label":   row["group_label"],
				"Order":   row["group_order"],
				"IsGroup": "Y",
				"Items":   []map[string]string{},
			}
		}

		key := row["item_group"]
		var item map[string]string

		if !util.EmptyBool(isOutRole) {
			item = map[string]string{
				"ID":    row["item_id"],
				"Label": row["item_label"],
				"Href":  row["item_href"],
				"Order": row["item_order"],
				"Role":  row["item_roles"],
			}
		} else {
			item = map[string]string{
				"ID":    row["item_id"],
				"Label": row["item_label"],
				"Href":  row["item_href"],
				"Order": row["item_order"],
			}
		}

		groupMap[key]["Items"] = append(groupMap[key]["Items"].([]map[string]string), item)

		// 중복된 순서 방지
		if !orderMap[key] {
			orderList = append(orderList, key)
			orderMap[key] = true
		}
	}

	// 결과 정렬 순서대로 재구성
	result := make([]map[string]interface{}, 0, len(orderList))
	for _, key := range orderList {
		if len(groupMap[key]["Items"].([]map[string]string)) > 0 {
			result = append(result, groupMap[key])
		}
	}

	return result
}

func contains(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func SetCookie(c *gin.Context, key string, val string, time int) {
	c.SetCookie(
		key,  // 쿠키 이름
		val,  // 값
		time, // max-age(초)
		"/",  // path
		"",   // domain (빈 문자열이면 Host 도메인)
		true, // secure (https 전용)
		true, // httpOnly
	)
}
