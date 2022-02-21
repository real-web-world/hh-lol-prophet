package hh_lol_prophet

import (
	"github.com/gin-gonic/gin"

	"github.com/real-web-world/hh-lol-prophet/api"
	"github.com/real-web-world/hh-lol-prophet/services/ws"
)

func RegisterRoutes(r *gin.Engine) {
	r.Any("test", api.DevHand)
	r.GET("ws", func(c *gin.Context) {
		ws.ServeWs(ws.ServerHub, c.Writer, c.Request)
	})
}
