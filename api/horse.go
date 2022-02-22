package api

import (
	"strings"

	"github.com/gin-gonic/gin"

	ginApp "github.com/real-web-world/hh-lol-prophet/pkg/gin"
	"github.com/real-web-world/hh-lol-prophet/services/lcu"
)

type (
	summonerNameReq struct {
		SummonerName string `json:"summonerName"`
	}
)

func QueryHorseBySummonerName(c *gin.Context) {
	app := ginApp.GetApp(c)
	d := &summonerNameReq{}
	if err := c.ShouldBind(d); err != nil {
		app.ValidError(err)
		return
	}
	summonerName := strings.TrimSpace(d.SummonerName)
	if summonerName == "" {
		app.ErrorMsg("名称必填")
		return
	}
	info, err := lcu.QuerySummonerByName(d.SummonerName)
	if err != nil {
		app.CommonError(err)
		return
	}
	app.Data(info)
}

func CopyHorseMsgToClipBoard(c *gin.Context) {
	app := ginApp.GetApp(c)
	app.Success()
}
