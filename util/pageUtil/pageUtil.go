package pageUtil

import (
	"encoding/json"
	"fmt"
	"gin_starter/model/core"
	"gin_starter/util"
	"gin_starter/util/auth"
	"log"
	"net/http"
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

// 로그인 체크, 토큰 체크, 만료시 쿠키갱신, 메뉴리턴
func RenderPageCheckLogin(c *gin.Context, isCheckLogin bool) []map[string]interface{} {
	if !util.EmptyBool(isCheckLogin) {

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

			// data["NEW_AT"] = newAT
			// data["NEW_RT"] = newRT

			SetCookie(c, "acc_token", newAT, 60*15)
			SetCookie(c, "ref_token", newRT, 60*60*24*7)
		}

		result, err := core.BuildSelectQuery(c, nil, "select u_auth_type, u_auth_level from _user where u_id = ? AND u_auth_type != 'U' ", []string{claims.JWTUserID}, "JWTAuthMiddleware.err")
		if err != nil {
			c.Redirect(http.StatusFound, "/adm/manage/login")
			c.Abort()
			return nil
		}

		c.Set("user_id", claims.JWTUserID)
		c.Set("user_type", result[0]["u_auth_type"])
		c.Set("user_level", result[0]["u_auth_level"])

		resultMenu, err := core.BuildSelectQuery(c, nil, `
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
			LEFT JOIN _menu_groups mg ON mg.mg_idx = mi.mi_group_id
			ORDER BY IFNULL(mg.mg_order, mi.mi_order), mi.mi_order`, []string{}, "get Menu sql err")
		if err != nil {
			c.Redirect(http.StatusFound, "/adm/manage/login")
			c.Abort()
			return nil
		}

		return FilterMenusByRole(resultMenu, result[0]["u_auth_type"])
	}

	return nil
}

// 페이지 출력
func RenderPage(c *gin.Context, page string, customData gin.H) {

	data := gin.H{
		"IsLoggedIn": false,
		"UserName":   "",
		"ShowFooter": true,
		"Menus":      []map[string]interface{}{},
	}

	for k, v := range customData {
		data[k] = v
	}

	tmpl, err := template.ParseFiles(
		"templates/layouts/layout.tmpl",
		fmt.Sprintf("templates/pages/%s.tmpl", page),
		"templates/components/navbar.tmpl",
		"templates/components/sidebar.tmpl",
		"templates/components/footer.tmpl",
	)
	if err != nil {
		log.Fatalf("[종료] 템플릿 로딩 실패: %v", err)
	}

	// log.Println("RenderPage ")
	// log.Println(data)

	c.Status(http.StatusOK)
	tmpl.ExecuteTemplate(c.Writer, "layout", data)
}

func FilterMenusByRole(data []map[string]string, userRole string) []map[string]interface{} {

	groupMap := map[string]map[string]interface{}{}
	orderList := []string{}

	// 1차 그룹생성
	for _, row := range data {
		groupID := row["group_id"]
		itemOrder := row["item_order"]

		if _, exists := groupMap[groupID]; !exists {
			if !util.EmptyString(groupID) {
				groupMap[groupID] = map[string]interface{}{
					"ID":    groupID,
					"Label": row["group_label"],
					"Order": row["group_order"],
					"Items": []map[string]string{},
				}
				orderList = append(orderList, groupID)
			} else {
				orderList = append(orderList, itemOrder)
			}
		}
	}

	for _, row := range data {
		itemOrder := row["item_order"]
		roles := []string{}
		if err := json.Unmarshal([]byte(row["item_roles"]), &roles); err != nil {
			continue
		}

		if !contains(roles, userRole) {
			continue
		}

		item := map[string]string{
			"ID":    row["item_id"],
			"Label": row["item_label"],
			"Href":  row["item_href"],
			"Order": row["item_order"],
		}

		groupID := row["item_group"]
		if group, exists := groupMap[groupID]; exists {
			group["Items"] = append(group["Items"].([]map[string]string), item)
		} else {
			groupMap[itemOrder] = map[string]interface{}{
				"ID":    itemOrder,
				"Label": "",
				"Order": itemOrder,
				"Items": []map[string]string{{
					"ID":    row["item_id"],
					"Label": row["item_label"],
					"Href":  row["item_href"],
					"Order": itemOrder,
				}},
			}
		}
	}

	var result []map[string]interface{}

	for _, id := range orderList {
		if group, exists := groupMap[id]; exists {
			if len(group["Items"].([]map[string]string)) > 0 {
				result = append(result, group)
			}
		}
	}

	// log.Println("FilterMenusByRole END ")
	// log.Println(result)

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
