package adm

import (
	"gin_starter/model/core"
	"gin_starter/util"
	"gin_starter/util/pageUtil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupAdminChatApiRoutes(rg *gin.RouterGroup) {

	adminChatApiGroup := rg.Group("/api")
	{
		adminChatApiGroup.GET("/chat/history", func(c *gin.Context) { getChatHistory(c) })
		adminChatApiGroup.POST("/chat/message", func(c *gin.Context) { postChatMessage(c) })
	}
}

func getChatHistory(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "", 0)

	getData := util.GetFields(c, map[string][2]string{
		"room": {"room", "0"},
	})

	history, _ := core.BuildSelectQuery(c, nil, `
        SELECT cm_idx id, cm_room_id room_id, cm_sender_id sender_id, cm_receiver_id receiver_id, cm_content content, cm_timestamp timestamp
        FROM _chat_messages
        WHERE cm_room_id = ?
        ORDER BY cm_timestamp ASC
    `, []string{getData["room"]}, "getChatHistory")

	c.JSON(http.StatusOK, gin.H{"data": history})
}
func postChatMessage(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "", 0)

	postData := util.PostFields(c, map[string][2]string{
		"room_id":     {"cm_room_id", ""},
		"sender_id":   {"cm_sender_id", ""},
		"receiver_id": {"cm_receiver_id", ""},
		"content":     {"cm_content", ""},
	})

	if !util.EmptyString(postData["cm_room_id"]) {
		lastID, _ := core.BuildInsertQuery(c, nil, "_chat_messages", map[string]string{
			"cm_room_id":     postData["cm_room_id"],
			"cm_sender_id":   postData["cm_sender_id"],
			"cm_receiver_id": postData["cm_receiver_id"],
			"cm_content":     postData["cm_content"],
		}, "postChatMessage")

		c.JSON(http.StatusOK, gin.H{
			"message":   "채팅이 저장되었습니다",
			"insert_id": lastID,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message":   "저장에 실패했습니다",
			"insert_id": "",
		})
	}

}
