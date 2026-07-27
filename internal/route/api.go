package route

import (
	"net/http"

	"github.com/real-web-world/hh-lol-prophet/internal/app"
	api "github.com/real-web-world/hh-lol-prophet/internal/handler"
)

func initAPIRoutes(r *http.ServeMux, inst *app.App) error {
	var err error
	initDevModule(r, inst)
	if err = initV1Module(r, inst); err != nil {
		return err
	}
	return nil
}

func initV1Module(r *http.ServeMux, inst *app.App) error {
	apiH := inst.GetProphet().Api()
	// 查询用户马匹信息
	r.Handle("POST /v1/horse/queryBySummonerName",
		apiH.ProphetActiveMid(http.HandlerFunc(apiH.QueryHorseBySummonerName)))
	// 获取所有配置
	r.HandleFunc("POST /v1/config/getAll", apiH.GetAllConf)
	// 更新配置
	r.HandleFunc("POST /v1/config/update", apiH.UpdateClientConf)
	// 获取lcu认证信息
	r.HandleFunc("POST /v1/lcu/getAuthInfo", apiH.GetLcuAuthInfo)
	// 获取app信息
	r.HandleFunc("POST /v1/app/getInfo", apiH.GetAppInfo)
	// 复制马匹信息到剪切板
	r.HandleFunc("POST /v1/horse/copyHorseMsgToClipBoard", apiH.CopyHorseMsgToClipBoard)
	// lcu proxy
	r.HandleFunc("/v1/lcu/proxy/{path...}", apiH.LcuProxy)
	return nil
}

func initDevModule(r *http.ServeMux, _ *app.App) {
	appH := api.NewApp()
	r.HandleFunc("/test", appH.DevHand)
}
