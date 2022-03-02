package hh_lol_prophet

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/avast/retry-go"
	"github.com/getsentry/sentry-go"
	sentryGin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/real-web-world/hh-lol-prophet/global"
	ginApp "github.com/real-web-world/hh-lol-prophet/pkg/gin"
	"github.com/real-web-world/hh-lol-prophet/services/lcu"
	"github.com/real-web-world/hh-lol-prophet/services/lcu/models"
	"github.com/real-web-world/hh-lol-prophet/services/logger"
)

type (
	lcuWsEvt  string
	GameState string
	Prophet   struct {
		ctx          context.Context
		opts         *options
		httpSrv      *http.Server
		lcuPort      int
		lcuToken     string
		lcuActive    bool
		currSummoner *lcu.CurrSummoner
		cancel       func()
		api          *Api
		mu           *sync.Mutex
		GameState    GameState
	}
	wsMsg struct {
		Data      interface{} `json:"data"`
		EventType string      `json:"event_type"`
		Uri       string      `json:"uri"`
	}
	options struct {
		debug       bool
		enablePprof bool
		httpAddr    string
	}
)

const (
	onJsonApiEventPrefixLen              = len(`[8,"OnJsonApiEvent",`)
	gameFlowChangedEvt          lcuWsEvt = "/lol-gameflow/v1/gameflow-phase"
	champSelectUpdateSessionEvt lcuWsEvt = "/lol-champ-select/v1/session"
)

// gameState
const (
	GameStateNone        GameState = "none"
	GameStateChampSelect GameState = "champSelect"
	GameStateReadyCheck  GameState = "ReadyCheck"
	GameStateInGame      GameState = "inGame"
	GameStateOther       GameState = "other"
)
const (
	acpGBK = 936
)

var (
	defaultOpts = &options{
		debug:       false,
		enablePprof: true,
		httpAddr:    ":4396",
	}
	errWebviewQuit = errors.New("webview quit")
)

func NewProphet(opts ...ApplyOption) *Prophet {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Prophet{
		ctx:       ctx,
		cancel:    cancel,
		mu:        &sync.Mutex{},
		opts:      defaultOpts,
		GameState: GameStateNone,
	}
	if global.IsDevMode() {
		opts = append(opts, WithDebug())
	} else {
		opts = append(opts, WithProd())
	}
	p.api = &Api{p: p}
	for _, fn := range opts {
		fn(p.opts)
	}
	return p
}
func (p *Prophet) Run() error {
	go p.MonitorStart()
	go p.captureStartMessage()
	p.initGin()
	go p.initWebview()
	log.Printf("%s已启动 v%s -- %s", global.AppName, APPVersion, global.WebsiteTitle)
	return p.notifyQuit()
}
func (p *Prophet) isLcuActive() bool {
	return p.lcuActive
}
func (p *Prophet) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	// stop all task
	return nil
}
func (p *Prophet) MonitorStart() {
	for {
		if !p.isLcuActive() {
			port, token, err := lcu.GetLolClientApiInfo()
			if err != nil {
				if !errors.Is(lcu.ErrLolProcessNotFound, err) {
					logger.Error("获取lcu info 失败", zap.Error(err))
				}
				time.Sleep(time.Second)
				continue
			}
			p.initLcuClient(port, token)
			err = p.initGameFlowMonitor(port, token)
			if err != nil {
				logger.Error("游戏流程监视器 err:", err)
			}
			p.lcuActive = false
			p.currSummoner = nil
		}
		time.Sleep(time.Second)
	}
}

func (p *Prophet) notifyQuit() error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	g, c := errgroup.WithContext(p.ctx)
	// http
	g.Go(func() error {
		err := p.httpSrv.ListenAndServe()
		if err != nil || !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	// http-shutdown
	g.Go(func() error {
		<-c.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		return p.httpSrv.Shutdown(ctx)
	})
	// wait quit
	g.Go(func() error {
		for {
			select {
			case <-p.ctx.Done():
				return p.ctx.Err()
			case <-interrupt:
				_ = p.Stop()
			}
		}
	})
	err := g.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
func (p *Prophet) initLcuClient(port int, token string) {
	lcu.InitCli(port, token)
}
func (p *Prophet) initGameFlowMonitor(port int, authPwd string) error {
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	rawUrl := fmt.Sprintf("wss://127.0.0.1:%d/", port)
	header := http.Header{}
	authSecret := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("riot:%s", authPwd)))
	header.Set("Authorization", "Basic "+authSecret)
	u, _ := url.Parse(rawUrl)
	logger.Debug(fmt.Sprintf("connect to lcu %s", u.String()))
	c, _, err := dialer.Dial(u.String(), header)
	if err != nil {
		logger.Error("连接到lcu ws 失败", zap.Error(err))
		return err
	}
	defer c.Close()
	err = retry.Do(func() error {
		currSummoner, err := lcu.GetCurrSummoner()
		if err == nil {
			p.currSummoner = currSummoner
		}
		return err
	}, retry.Attempts(5), retry.Delay(time.Second))
	if err != nil {
		return errors.New("获取当前召唤师信息失败:" + err.Error())
	}
	p.lcuActive = true
	// if global.IsDevMode() {
	// 	p.ChampionSelectStart()
	// }

	_ = c.WriteMessage(websocket.TextMessage, []byte("[5, \"OnJsonApiEvent\"]"))
	for {
		msgType, message, err := c.ReadMessage()
		if err != nil {
			// log.Println("read:", err)
			logger.Error("lol事件监控读取消息失败", zap.Error(err))
			return err
		}
		msg := &wsMsg{}
		if msgType != websocket.TextMessage || len(message) < onJsonApiEventPrefixLen+1 {
			continue
		}
		_ = json.Unmarshal(message[onJsonApiEventPrefixLen:len(message)-1], msg)
		// log.Println("ws evt: ", msg.Uri)
		switch msg.Uri {
		case string(gameFlowChangedEvt):
			gameFlow, ok := msg.Data.(string)
			if !ok {
				continue
			}
			p.onGameFlowUpdate(gameFlow)
		case string(champSelectUpdateSessionEvt):
			bts, err := json.Marshal(msg.Data)
			if err != nil {
				continue
			}
			sessionInfo := &lcu.ChampSelectSessionInfo{}
			err = json.Unmarshal(bts, sessionInfo)
			if err != nil {
				log.Println("解析结构体失败", err)
				continue
			}
			go func() {
				_ = p.onChampSelectSessionUpdate(sessionInfo)
			}()
		default:

		}

		// log.Printf("recv: %s", message)
	}
}
func (p *Prophet) onGameFlowUpdate(gameFlow string) {
	// clientCfg := global.GetClientConf()
	logger.Debug("切换状态:" + gameFlow)
	switch gameFlow {
	case string(models.GameFlowChampionSelect):
		log.Println("进入英雄选择阶段,正在计算用户分数")
		sentry.CaptureMessage("进入英雄选择阶段,正在计算用户分数")
		p.updateGameState(GameStateChampSelect)
		go p.ChampionSelectStart()
	case string(models.GameFlowNone):
		p.updateGameState(GameStateNone)
	case string(models.GameFlowInProgress):
		p.updateGameState(GameStateInGame)
		go p.CalcEnemyTeamScore()
	case string(models.GameFlowReadyCheck):
		p.updateGameState(GameStateReadyCheck)
		clientCfg := global.GetClientConf()
		if clientCfg.AutoAcceptGame {
			go p.AcceptGame()
		}
	default:
		p.updateGameState(GameStateOther)
	}

}
func (p *Prophet) updateGameState(state GameState) {
	p.mu.Lock()
	p.GameState = state
	p.mu.Unlock()
}
func (p *Prophet) getGameState() GameState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.GameState
}
func (p *Prophet) captureStartMessage() {
	for i := 0; i < 5; i++ {
		if global.GetUserInfo().IP != "" {
			break
		}
		time.Sleep(time.Second * 2)
	}
	sentry.CaptureMessage(global.AppName + "已启动")
}
func (p *Prophet) initGin() {
	if p.opts.debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.LoggerWithFormatter(ginApp.LogFormatter))
	if p.opts.enablePprof {
		pprof.RouteRegister(engine.Group(""))
	}
	engine.Use(ginApp.PrepareProc)
	engine.Use(sentryGin.New(sentryGin.Options{
		Repanic: true,
		Timeout: 3 * time.Second,
	}))
	engine.Use(ginApp.Cors())
	engine.Use(ginApp.ErrHandler)
	RegisterRoutes(engine, p.api)

	srv := &http.Server{
		Addr:    p.opts.httpAddr,
		Handler: engine,
	}
	p.httpSrv = srv
}
func (p *Prophet) initWebview() {
	clientCfg := global.GetClientConf()
	defaultUrl := "https://lol.buffge.com/dev/client?version=" + APPVersion
	websiteUrl := defaultUrl
	if clientCfg.ShouldAutoOpenBrowser != nil && !*clientCfg.ShouldAutoOpenBrowser {
		log.Println("自动打开浏览器选项已关闭,手动打开请访问 " + websiteUrl)
		return
	}
	// windowWeight := 1000
	// windowHeight := 650

	cmd := exec.Command("cmd", "/c", "start", websiteUrl)
	_ = cmd.Run()
	log.Println("界面已在浏览器中打开,若未打开请手动访问 " + websiteUrl)
	return
}
func (p Prophet) ChampionSelectStart() {
	clientCfg := global.GetClientConf()
	sendConversationMsgDelayCtx, cancel := context.WithTimeout(context.Background(),
		time.Second*time.Duration(clientCfg.ChooseChampSendMsgDelaySec))
	defer cancel()
	var conversationID string
	var summonerIDList []int64
	for i := 0; i < 3; i++ {
		time.Sleep(time.Second)
		// 获取队伍所有用户信息
		conversationID, summonerIDList, _ = getTeamUsers()
		if len(summonerIDList) != 5 {
			continue
		}
	}
	// if !false && global.IsDevMode() {
	// 	summonerIDList = []int64{2964390005, 4103784618, 4132401993, 4118593599, 4019221688}
	// 	// summonerIDList = []int64{4006944917}
	// }
	logger.Debug("队伍人员列表:", zap.Any("summonerIDList", summonerIDList))
	// 查询所有用户的信息并计算得分
	g := errgroup.Group{}
	summonerIDMapScore := map[int64]lcu.UserScore{}
	mu := sync.Mutex{}
	for _, summonerID := range summonerIDList {
		summonerID := summonerID
		g.Go(func() error {
			actScore, err := GetUserScore(summonerID)
			if err != nil {
				logger.Error("计算用户得分失败", zap.Error(err), zap.Int64("summonerID", summonerID))
				return nil
			}
			mu.Lock()
			summonerIDMapScore[summonerID] = *actScore
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	// 根据所有用户的分数判断小代上等马中等马下等马
	for _, score := range summonerIDMapScore {
		log.Printf("用户:%s,得分:%.2f\n", score.SummonerName, score.Score)
	}

	scoreCfg := global.GetScoreConf()
	allMsg := ""
	mergedMsg := ""
	// 发送到选人界面
	for _, scoreInfo := range summonerIDMapScore {
		var horse string
		horseIdx := 0
		for i, v := range scoreCfg.Horse {
			if scoreInfo.Score >= v.Score {
				horse = clientCfg.HorseNameConf[i]
				horseIdx = i
				break
			}
		}
		currKDASb := strings.Builder{}
		for i := 0; i < 5 && i < len(scoreInfo.CurrKDA); i++ {
			currKDASb.WriteString(fmt.Sprintf("%d/%d/%d  ", scoreInfo.CurrKDA[i][0], scoreInfo.CurrKDA[i][1],
				scoreInfo.CurrKDA[i][2]))
		}
		currKDAMsg := currKDASb.String()
		if len(currKDAMsg) > 0 {
			currKDAMsg = currKDAMsg[:len(currKDAMsg)-1]
		}
		msg := fmt.Sprintf("%s(%d): %s %s  -- %s", horse, int(scoreInfo.Score), scoreInfo.SummonerName,
			currKDAMsg, global.AdaptChatWebsiteTitle)
		<-sendConversationMsgDelayCtx.Done()
		if clientCfg.AutoSendTeamHorse {
			mergedMsg += msg + "\n"
		}
		if !clientCfg.AutoSendTeamHorse {
			if !scoreCfg.MergeMsg && !clientCfg.ShouldSendSelfHorse && p.currSummoner != nil &&
				scoreInfo.SummonerID == p.currSummoner.SummonerId {
				continue
			}
			allMsg += msg + "\n"
			mergedMsg += msg + "\n"
			continue
		}
		if !clientCfg.ShouldSendSelfHorse && p.currSummoner != nil &&
			scoreInfo.SummonerID == p.currSummoner.SummonerId {
			continue
		}
		if !clientCfg.ChooseSendHorseMsg[horseIdx] {
			continue
		}
		if scoreCfg.MergeMsg {
			continue
		}
		_ = SendConversationMsg(msg, conversationID)
		time.Sleep(time.Second)
	}
	if !clientCfg.AutoSendTeamHorse {
		log.Println("已将队伍马匹信息复制到剪切板")
		_ = clipboard.WriteAll(allMsg)
		return
	}
	if scoreCfg.MergeMsg {
		_ = SendConversationMsg(mergedMsg, conversationID)
	}
}
func (p Prophet) AcceptGame() {
	_ = lcu.AcceptGame()
}
func (p Prophet) CalcEnemyTeamScore() {
	// 获取当前游戏进程
	session, err := lcu.QueryGameFlowSession()
	if err != nil {
		return
	}
	if session.Phase != models.GameFlowInProgress {
		return
	}
	if p.currSummoner == nil {
		return
	}
	selfID := p.currSummoner.SummonerId
	selfTeamUsers, enemyTeamUsers := getAllUsersFromSession(selfID, session)
	_ = selfTeamUsers
	summonerIDList := enemyTeamUsers
	// if !false && global.IsDevMode() {
	// 	summonerIDList = []int64{2964390005, 4103784618, 4132401993, 4118593599, 4019221688}
	// 	// summonerIDList = []int64{4006944917}
	// }
	logger.Debug("敌方队伍人员列表:", zap.Any("summonerIDList", summonerIDList))
	if len(summonerIDList) == 0 {
		return
	}
	// 查询所有用户的信息并计算得分
	g := errgroup.Group{}
	summonerIDMapScore := map[int64]lcu.UserScore{}
	mu := sync.Mutex{}
	for _, summonerID := range summonerIDList {
		summonerID := summonerID
		g.Go(func() error {
			actScore, err := GetUserScore(summonerID)
			if err != nil {
				logger.Error("计算用户得分失败", zap.Error(err), zap.Int64("summonerID", summonerID))
				return nil
			}
			mu.Lock()
			summonerIDMapScore[summonerID] = *actScore
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	// 根据所有用户的分数判断小代上等马中等马下等马
	for _, score := range summonerIDMapScore {
		currKDASb := strings.Builder{}
		for i := 0; i < 5 && i < len(score.CurrKDA); i++ {
			currKDASb.WriteString(fmt.Sprintf("%d/%d/%d  ", score.CurrKDA[i][0], score.CurrKDA[i][1],
				score.CurrKDA[i][2]))
		}
		currKDAMsg := currKDASb.String()
		log.Printf("敌方用户:%s,得分:%.2f,kda:%s\n", score.SummonerName, score.Score, currKDAMsg)
	}
	clientCfg := global.GetClientConf()
	scoreCfg := global.GetScoreConf()
	allMsg := ""
	// 发送到选人界面
	for _, scoreInfo := range summonerIDMapScore {
		time.Sleep(time.Second / 2)
		var horse string
		// horseIdx := 0
		for i, v := range scoreCfg.Horse {
			if scoreInfo.Score >= v.Score {
				horse = clientCfg.HorseNameConf[i]
				// horseIdx = i
				break
			}
		}
		currKDASb := strings.Builder{}
		for i := 0; i < 5 && i < len(scoreInfo.CurrKDA); i++ {
			currKDASb.WriteString(fmt.Sprintf("%d/%d/%d  ", scoreInfo.CurrKDA[i][0], scoreInfo.CurrKDA[i][1],
				scoreInfo.CurrKDA[i][2]))
		}
		currKDAMsg := currKDASb.String()
		if len(currKDAMsg) > 0 {
			currKDAMsg = currKDAMsg[:len(currKDAMsg)-1]
		}
		msg := fmt.Sprintf("%s(%d): %s %s  -- %s", horse, int(scoreInfo.Score), scoreInfo.SummonerName,
			currKDAMsg, global.AdaptChatWebsiteTitle)
		allMsg += msg + "\n"
	}
	_ = clipboard.WriteAll(allMsg)
}

func (p Prophet) onChampSelectSessionUpdate(sessionInfo *lcu.ChampSelectSessionInfo) error {
	isSelfPick := false
	isSelfBan := false
	userActionID := 0
	if len(sessionInfo.Actions) == 0 {
		return nil
	}
loop:
	for _, actions := range sessionInfo.Actions {
		for _, action := range actions {
			if action.ActorCellId == sessionInfo.LocalPlayerCellId && action.IsInProgress {
				userActionID = action.Id
				if action.Type == lcu.ChampSelectPatchTypePick {
					isSelfPick = true
					break loop
				} else if action.Type == lcu.ChampSelectPatchTypeBan {
					isSelfBan = true
					break loop
				}
			}
		}
	}
	clientCfg := global.GetClientConf()
	if clientCfg.AutoPickChampID > 0 && isSelfPick {
		_ = lcu.PickChampion(clientCfg.AutoPickChampID, userActionID)
	}
	if clientCfg.AutoBanChampID > 0 && isSelfBan {
		_ = lcu.BanChampion(clientCfg.AutoBanChampID, userActionID)
	}
	return nil
}
