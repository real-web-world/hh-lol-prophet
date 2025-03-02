package hh_lol_prophet

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"time"

	"github.com/real-web-world/hh-lol-prophet/conf"
	"github.com/real-web-world/hh-lol-prophet/global"
	ginApp "github.com/real-web-world/hh-lol-prophet/pkg/gin"
	"github.com/real-web-world/hh-lol-prophet/services/db/models"
	"github.com/real-web-world/hh-lol-prophet/services/lcu"
)

type (
	Api struct {
		p *Prophet
	}
	summonerNameReq struct {
		SummonerName string `json:"summonerName"`
	}
)

func (api Api) ProphetActiveMid(c *gin.Context) {
	app := ginApp.GetApp(c)
	if !api.p.lcuActive {
		app.ErrorMsg("请检查lol客户端是否已启动")
		return
	}
	c.Next()
}
func (api Api) QueryHorseBySummonerName(c *gin.Context) {
	app := ginApp.GetApp(c)
	d := &summonerNameReq{}
	if err := c.ShouldBind(d); err != nil {
		app.ValidError(err)
		return
	}
	summonerName := strings.TrimSpace(d.SummonerName)
	var summoner *lcu.Summoner
	if summonerName == "" {
		if api.p.currSummoner == nil {
			app.ErrorMsg("系统错误")
			return
		}
		summoner = lcu.ConvertCurrSummonerToSummoner(api.p.currSummoner)
	} else {
		info, err := lcu.QuerySummonerByName(summonerName)
		if err != nil || info.SummonerId <= 0 {
			app.ErrorMsg("未查询到召唤师")
			return
		}
		summoner = info
	}
	scoreInfo, err := GetUserScore(summoner)
	if err != nil {
		app.CommonError(err)
		return
	}
	scoreCfg := global.GetScoreConf()
	clientCfg := global.GetClientConf()
	var horse string
	for i, v := range scoreCfg.Horse {
		if scoreInfo.Score >= v.Score {
			horse = clientCfg.HorseNameConf[i]
			break
		}
	}
	app.Data(gin.H{
		"score":   scoreInfo.Score,
		"currKDA": scoreInfo.CurrKDA,
		"horse":   horse,
	})
}

func (api Api) CopyHorseMsgToClipBoard(c *gin.Context) {
	app := ginApp.GetApp(c)
	app.Success()
}
func (api Api) DelAllCurrSummonerFriends(c *gin.Context) {
	app := ginApp.GetApp(c)
	if !api.p.lcuActive {
		app.ErrorMsg("请检查lol客户端是否已启动")
		return
	}
	if lcu.DelAllCurrSummonerFriendsStart {
		app.String("删除好友正在进行中...\n请耐心等待，请勿重复提交")
		return
	}
	t := time.Now().Unix()
	t1 := strconv.FormatInt(t, 10)
	app.String("正在删除全部好友....\n请在exe界面查看进度....\n你拥有30秒钟时间考虑，如果反悔可以立即关闭exe程序.\n\n你当前所有好友备份在\n C:\\好友最后的备份" + t1 + ".txt")
	go lcu.DelAllCurrSummonerFriends(t1)
}
func (api Api) GetAllConf(c *gin.Context) {
	app := ginApp.GetApp(c)
	app.Data(global.GetClientConf())
}
func (api Api) UpdateClientConf(c *gin.Context) {
	app := ginApp.GetApp(c)
	d := &conf.UpdateClientConfReq{}
	if err := c.ShouldBind(d); err != nil {
		app.ValidError(err)
		return
	}
	cfg := global.SetClientConf(*d)
	bts, _ := json.Marshal(cfg)
	m := models.Config{}
	err := m.Update(models.LocalClientConfKey, string(bts))
	if err != nil {
		app.CommonError(err)
		return
	}
	app.Success()
}
func (api Api) DevHand(c *gin.Context) {
	app := ginApp.GetApp(c)
	app.Data(gin.H{
		"buffge": 23456,
	})
}
func (api Api) GetAppInfo(c *gin.Context) {
	app := ginApp.GetApp(c)
	app.Data(global.AppBuildInfo)
}
func (api Api) GetLcuAuthInfo(c *gin.Context) {
	app := ginApp.GetApp(c)
	port, token, err := lcu.GetLolClientApiInfo()
	if err != nil {
		app.CommonError(err)
		return
	}
	app.Data(gin.H{
		"token": token,
		"port":  port,
	})
}
