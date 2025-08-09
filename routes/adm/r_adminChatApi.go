package adm

import (
	"gin_starter/model/dbCore"
	"gin_starter/util/utilCore"
	"gin_starter/util/utilCore/pageUtil"
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

	rulesGet := []utilCore.InputRule{
		{InputKey: "room", OutputKey: "room", Label: "ID",
			Allow: utilCore.AllowNumber, Required: true, Default: "0", TrimSpace: true},
	}
	getData, ok := utilCore.BindAutoAndRespond(c, rulesGet, "getChatHistory BindAutoAndRespond")
	if !ok {
		return
	}

	history, _ := dbCore.BuildSelectQuery(c, nil, `
        SELECT cm_idx id, cm_room_id room_id, cm_sender_id sender_id, cm_receiver_id receiver_id, cm_content content, cm_timestamp timestamp
        FROM _chat_messages
        WHERE cm_room_id = ?
        ORDER BY cm_timestamp ASC
    `, []string{getData["room"]}, "getChatHistory")

	c.JSON(http.StatusOK, gin.H{"data": history})
}
func postChatMessage(c *gin.Context) {

	pageUtil.RenderPageCheckLogin(c, "", 0)

	rules := []utilCore.InputRule{
		{InputKey: "room_id", OutputKey: "cm_room_id", Label: "방 ID",
			Allow: utilCore.AllowKorEngNumSp, Required: true, MinLen: 3, MaxLen: 100, TrimSpace: true},
		{InputKey: "sender_id", OutputKey: "cm_sender_id", Label: "발송자",
			Allow: utilCore.AllowKorEngNum, Required: true, MinLen: 1, MaxLen: 50, TrimSpace: true},
		{InputKey: "receiver_id", OutputKey: "cm_receiver_id", Label: "수신자",
			Allow: utilCore.AllowKorEngNum, Required: true, MinLen: 2, MaxLen: 50, TrimSpace: true},
		{InputKey: "content", OutputKey: "cm_content", Label: "내용",
			Allow: utilCore.AllowSafeText, Required: true, MinLen: 1, TrimSpace: true},
	}
	postData, ok := utilCore.BindAutoAndRespond(c, rules, "postChatMessage BindAutoAndRespond")
	if !ok {
		return
	}

	if !utilCore.EmptyString(postData["cm_room_id"]) {
		lastID, _ := dbCore.BuildInsertQuery(c, nil, "_chat_messages", map[string]string{
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
