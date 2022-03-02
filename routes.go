package hh_lol_prophet

import (
	"github.com/gin-gonic/gin"

	"github.com/real-web-world/hh-lol-prophet/services/ws"
)

func RegisterRoutes(r *gin.Engine, api *Api) {
	r.Any("test", api.DevHand)
	r.GET("ws", func(c *gin.Context) {
		ws.ServeWs(ws.ServerHub, c.Writer, c.Request)
	})
	initV1Module(r, api)
}

func initV1Module(r *gin.Engine, api *Api) {
	v1 := r.Group("v1")
	// 查询用户马匹信息
	v1.POST("horse/queryBySummonerName", api.ProphetActiveMid, api.QueryHorseBySummonerName)
	// 获取所有配置
	v1.POST("config/getAll", api.GetAllConf)
	// 更新配置
	v1.POST("config/update", api.UpdateClientConf)
	// 获取lcu认证信息
	v1.POST("lcu/getAuthInfo", api.GetLcuAuthInfo)
	// 获取app信息
	v1.POST("app/getInfo", api.GetAppInfo)
	// 复制马匹信息到剪切板
	v1.POST("horse/copyHorseMsgToClipBoard", api.CopyHorseMsgToClipBoard)
}
