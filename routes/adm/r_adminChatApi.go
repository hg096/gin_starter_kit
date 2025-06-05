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
        SELECT id, room_id, sender_id, receiver_id, content, timestamp
        FROM chat_messages
        WHERE room_id = ?
        ORDER BY timestamp ASC
    `, []string{getData["room"]}, "getChatHistory")

	c.JSON(http.StatusOK, gin.H{"data": history})
}
func postChatMessage(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "", 0)

	postData := util.PostFields(c, map[string][2]string{
		"room_id":     {"room_id", ""},
		"sender_id":   {"sender_id", ""},
		"receiver_id": {"receiver_id", ""},
		"content":     {"content", ""},
	})

	if !util.EmptyString(postData["room_id"]) {
		lastID, _ := core.BuildInsertQuery(c, nil, "chat_messages", map[string]string{
			"room_id":     postData["room_id"],
			"sender_id":   postData["sender_id"],
			"receiver_id": postData["receiver_id"],
			"content":     postData["content"],
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
