package prophet

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	appProj "github.com/real-web-world/hh-lol-prophet"
	"github.com/real-web-world/hh-lol-prophet/internal/config"
	"github.com/real-web-world/hh-lol-prophet/pkg/lcu"
	lcuModels "github.com/real-web-world/hh-lol-prophet/pkg/lcu/model"
)

type (
	Api struct {
		p *Prophet
		// calcScoreCfg  *config.CalcScoreConf
		// clientUserCfg *config.ClientUserConf
	}
	summonerNameReq struct {
		SummonerName string `json:"summonerName"`
	}
	Response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data any    `json:"data"`
	}
)

func (api Api) ProphetActiveMid(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !api.p.lcuActive {
			_, _ = w.Write([]byte("请检查lol客户端是否已启动"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (api Api) QueryHorseBySummonerName(w http.ResponseWriter, r *http.Request) {
	d := &summonerNameReq{}
	bts, err := io.ReadAll(r.Body)
	if err != nil {
		_, _ = w.Write([]byte(`{"code":1,"msg":"服务器错误"}`))
	}
	defer func() {
		_ = r.Body.Close()
	}()
	_ = json.Unmarshal(bts, d)
	summonerName := strings.TrimSpace(d.SummonerName)
	var summoner *lcuModels.Summoner
	if summonerName == "" {
		if api.p.currSummoner == nil {
			_, _ = w.Write([]byte("系统错误"))
			return
		}
		summoner = lcu.ConvertCurrSummonerToSummoner(api.p.currSummoner)
	} else {
		info, err := api.p.lcu.QuerySummonerByName(summonerName)
		if err != nil || info.SummonerId <= 0 {
			_, _ = w.Write([]byte("未查询到召唤师"))
			return
		}
		summoner = info
	}
	scoreInfo, err := api.p.GetUserScore(summoner)
	if err != nil {
		commonError(w, err)
		return
	}
	scoreCfg := api.p.GetScoreConf()
	clientUserCfg := api.p.GetClientUserConf()
	var horse string
	for i, v := range scoreCfg.Horse {
		if scoreInfo.Score >= v.Score {
			horse = clientUserCfg.HorseNameConf[i]
			break
		}
	}
	commonData(w, map[string]any{
		"score":   scoreInfo.Score,
		"currKDA": scoreInfo.CurrKDA,
		"horse":   horse,
	})
}

func (api Api) CopyHorseMsgToClipBoard(w http.ResponseWriter, r *http.Request) {
	commonOk(w)
}

func (api Api) GetAllConf(w http.ResponseWriter, r *http.Request) {
	commonData(w, api.p.GetClientUserConf())
}

func (api Api) UpdateClientConf(w http.ResponseWriter, r *http.Request) {
	d := &config.UpdateClientUserConfReq{}
	_, err := api.p.SetClientUserConf(*d)
	if err != nil {
		api.p.l.Debug("设置客户端配置失败", slog.Any("err", err))
		commonErrorMsg(w, "系统错误,更新配置失败")
	}
	commonOk(w)
}

func (api Api) DevHand(w http.ResponseWriter, r *http.Request) {
	commonData(w, map[string]any{
		"buff": 23456,
	})
}

func (api Api) GetAppInfo(w http.ResponseWriter, r *http.Request) {
	commonData(w, map[string]any{
		"version":   appProj.APPVersion,
		"commit":    appProj.Commit,
		"buildUser": appProj.BuildUser,
		"buildTime": appProj.BuildTime,
	})
}

func (api Api) GetLcuAuthInfo(w http.ResponseWriter, r *http.Request) {
	port, token, err := lcu.GetLolClientApiInfo()
	if err != nil {
		commonError(w, err)
		return
	}
	commonData(w, map[string]any{
		"token": token,
		"port":  port,
	})
}

func (api Api) LcuProxy(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	r.URL.Path = path
	rp := api.p.lcuRP
	if rp == nil {
		commonErrorMsg(w, "反向代理未初始化")
		return
	}
	rp.ServeHTTP(w, r)
}

func commonData(w http.ResponseWriter, data any) {
	resp := &Response{
		Code: 0,
		Data: data,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func commonOk(w http.ResponseWriter) {
	_, _ = w.Write([]byte(`{"code":0}`))
}

func commonError(w http.ResponseWriter, err error) {
	_, _ = fmt.Fprintf(w, `{"code":1,"msg":"%v"}`, err)
}

func commonErrorMsg(w http.ResponseWriter, msg string) {
	_, _ = fmt.Fprintf(w, `{"code":1,"msg":"%s"}`, msg)
}
