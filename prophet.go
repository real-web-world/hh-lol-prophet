package hh_lol_prophet

import (
	"cmp"
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
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/avast/retry-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	bdkgin "github.com/real-web-world/bdk/gin"
	bdkmid "github.com/real-web-world/bdk/gin/middleware"

	"github.com/real-web-world/hh-lol-prophet/global"
	"github.com/real-web-world/hh-lol-prophet/services/lcu"
	"github.com/real-web-world/hh-lol-prophet/services/lcu/models"
	"github.com/real-web-world/hh-lol-prophet/services/logger"
)

type (
	GameState string
	Prophet   struct {
		ctx          context.Context
		opts         *options
		httpSrv      *http.Server
		lcuPort      int
		lcuToken     string
		lcuActive    bool
		currSummoner *models.SummonerProfileData
		cancel       func()
		api          *Api
		mu           *sync.Mutex
		GameState    GameState
		lcuRP        *lcu.RP
	}
	options struct {
		debug       bool
		enablePprof bool
		httpAddr    string
	}
)

// gameState
const (
	GameStateNone        GameState = "none"
	GameStateChampSelect GameState = "champSelect"
	GameStateReadyCheck  GameState = "ReadyCheck"
	GameStateInGame      GameState = "inGame"
	GameStateOther       GameState = "other"
	GameStateMatchmaking GameState = "Matchmaking"
)

var (
	defaultOpts = &options{
		debug:       false,
		enablePprof: true,
		httpAddr:    "127.0.0.1:4396",
	}
)
var (
	allowOriginRegex = regexp.MustCompile(".+?\\.buffge\\.com(:\\d+)?$")
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
	go p.initWebView()
	log.Printf("%s已启动 v%s -- %s", global.Conf.AppName, APPVersion, global.Conf.WebsiteTitle)
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
					logger.Warn("获取lcu info 失败", zap.Error(err))
				}
				time.Sleep(time.Second)
				continue
			}
			p.initLcuClient(port, token)
			err = p.initLcuRP(port, token)
			if err != nil {
				logger.Debug("初始化lcuRP失败", zap.Error(err))
			}
			err = p.initGameFlowMonitor(port, token)
			if err != nil {
				logger.Debug("游戏流程监视器 err:", zap.Error(err))
			}
			global.SetCurrSummoner(nil)
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
			_ = p.Stop()
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
func (p *Prophet) initLcuRP(port int, token string) error {
	rp, err := lcu.NewRP(port, token)
	if err == nil {
		p.lcuRP = rp
	}
	return err
}
func (p *Prophet) initGameFlowMonitor(port int, authPwd string) error {
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	rawUrl := lcu.GenerateClientWsUrl(port)
	header := http.Header{}
	authSecret := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", lcu.AuthUserName, authPwd)))
	header.Set("Authorization", "Basic "+authSecret)
	u, _ := url.Parse(rawUrl)
	c, _, err := dialer.Dial(u.String(), header)
	if err != nil {
		return err
	}
	logger.Debug(fmt.Sprintf("connect to lcu %s", u.String()))
	defer func() {
		_ = c.Close()
	}()
	err = retry.Do(func() error {
		currSummoner, err := lcu.GetSummonerProfile()
		if err == nil {
			p.currSummoner = currSummoner
		}
		return err
	}, retry.Attempts(5), retry.Delay(time.Second))
	if err != nil {
		return errors.New("获取当前召唤师信息失败:" + err.Error())
	}
	global.SetCurrSummoner(p.currSummoner)
	p.lcuActive = true
	_ = c.WriteMessage(websocket.TextMessage, lcu.SubscribeAllEventMsg)
	for {
		msgType, message, err := c.ReadMessage()
		if err != nil {
			logger.Debug("lol事件监控读取消息失败", zap.Error(err))
			return err
		}
		msg := &lcu.WsMsg{}
		if msgType != websocket.TextMessage || len(message) < lcu.OnJsonApiEventPrefixLen+1 {
			continue
		}
		_ = json.Unmarshal(message[lcu.OnJsonApiEventPrefixLen:len(message)-1], msg)
		switch msg.Uri {
		case string(lcu.WsEvtGameFlowChanged):
			gameFlow := string(msg.Data)
			p.onGameFlowUpdate(gameFlow)
		case string(lcu.WsEvtChampSelectUpdateSession):
			sessionInfo := &models.ChampSelectSessionInfo{}
			if err = json.Unmarshal(msg.Data, sessionInfo); err != nil {
				logger.Debug("champSelectUpdateSessionEvt 解析结构体失败", zap.Error(err))
				continue
			}
			go func() {
				_ = p.onChampSelectSessionUpdate(sessionInfo)
			}()
		default:

		}
	}
}
func (p *Prophet) onGameFlowUpdate(gameFlow string) {
	logger.Debug("切换状态:" + gameFlow)
	switch gameFlow {
	case string(models.GameFlowChampionSelect):
		logger.Info("进入英雄选择阶段,正在计算用户分数")
		p.updateGameState(GameStateChampSelect)
		go p.ChampionSelectStart()
	case string(models.GameFlowNone):
		p.updateGameState(GameStateNone)
	case string(models.GameFlowMatchmaking):
		p.updateGameState(GameStateMatchmaking)
	case string(models.GameFlowInProgress):
		p.updateGameState(GameStateInGame)
		go p.CalcEnemyTeamScore()
	case string(models.GameFlowReadyCheck):
		p.updateGameState(GameStateReadyCheck)
		clientCfg := global.GetClientUserConf()
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
	logger.Info(global.Conf.AppName + "已启动")
}
func (p *Prophet) initGin() {
	if p.opts.debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := bdkgin.NewGin()
	engine.Use(gin.LoggerWithFormatter(bdkgin.LogFormatter))
	if p.opts.enablePprof {
		pprof.RouteRegister(engine.Group(""))
	}
	engine.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			if global.IsDevMode() {
				return true
			}
			return allowOriginRegex.MatchString(origin)
		},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowWebSockets:  true,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	engine.Use(bdkmid.RecoveryWithLogFn(logger.Error))
	RegisterRoutes(engine, p.api)
	srv := &http.Server{
		Addr:    p.opts.httpAddr,
		Handler: engine,
	}
	p.httpSrv = srv
}
func (p *Prophet) initWebView() {
	clientCfg := global.GetClientUserConf()
	indexUrl := global.Conf.WebView.IndexUrl
	defaultUrl := indexUrl + "?version=" + APPVersion
	websiteUrl := defaultUrl
	if clientCfg.ShouldAutoOpenBrowser != nil && !*clientCfg.ShouldAutoOpenBrowser {
		log.Println("自动打开浏览器选项已关闭,手动打开请访问 " + websiteUrl)
		return
	}
	cmd := exec.Command("cmd", "/c", "start", websiteUrl)
	_ = cmd.Run()
	log.Println("界面已在浏览器中打开,若未打开请手动访问 " + websiteUrl)
	return
}
func (p *Prophet) ChampionSelectStart() {
	clientCfg := global.GetClientUserConf()
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
	//summonerIDList = []int64{2964390005, 4103784618, 4132401993, 4118593599, 4019221688}
	// 	// summonerIDList = []int64{4006944917}
	// }
	if len(summonerIDList) == 0 {
		return
	}
	logger.Debug("队伍人员列表:", zap.Any("summonerIDList", summonerIDList))
	// 查询所有用户的信息并计算得分
	g := errgroup.Group{}
	summonerScores := make([]*lcu.UserScore, 0, 5)
	mu := sync.Mutex{}
	summonerIDMapInfo, err := listSummoner(summonerIDList)
	if err != nil {
		logger.Error("查询召唤师信息失败", zap.Error(err), zap.Any("summonerIDList", summonerIDList))
		return
	}
	for _, summoner := range summonerIDMapInfo {
		summoner := summoner
		summonerID := summoner.SummonerId
		g.Go(func() error {
			actScore, err := GetUserScore(summoner)
			if err != nil {
				logger.Error("计算用户得分失败", zap.Error(err), zap.Int64("summonerID", summonerID))
				return nil
			}
			mu.Lock()
			summonerScores = append(summonerScores, actScore)
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	slices.SortFunc(summonerScores, func(a, b *lcu.UserScore) int {
		return cmp.Compare(b.Score, a.Score)
	})
	// 根据所有用户的分数判断小代上等马中等马下等马
	//for _, score := range summonerIDMapScore {
	//	fmt.Printf("用户:%s,得分:%.2f\n", score.SummonerName, score.Score)
	//}
	scoreCfg := global.GetScoreConf()
	allMsg := ""
	mergedMsg := ""
	// 发送到选人界面
	for _, scoreInfo := range summonerScores {
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
		msg := fmt.Sprintf("%s(%d): %s %s", horse, int(scoreInfo.Score), scoreInfo.SummonerName,
			currKDAMsg)
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
		time.Sleep(time.Millisecond * 2100)
	}
	if !clientCfg.AutoSendTeamHorse {
		_ = clipboard.WriteAll(allMsg)
		fmt.Println("已将队伍马匹信息复制到剪切板 ", time.Now().Format(time.DateTime))
		fmt.Println()
		fmt.Println(allMsg)
		return
	}
	if scoreCfg.MergeMsg {
		_ = SendConversationMsg(mergedMsg, conversationID)
	}
}
func (p *Prophet) AcceptGame() {
	_ = lcu.AcceptGame()
}
func (p *Prophet) CalcEnemyTeamScore() {
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
	summonerScores := make([]*lcu.UserScore, 0, 5)
	mu := sync.Mutex{}
	summonerIDMapInfo, err := listSummoner(summonerIDList)
	if err != nil {
		logger.Error("查询召唤师信息失败", zap.Error(err), zap.Any("summonerIDList", summonerIDList))
		return
	}
	for _, summoner := range summonerIDMapInfo {
		summoner := summoner
		summonerID := summoner.SummonerId
		g.Go(func() error {
			actScore, err := GetUserScore(summoner)
			if err != nil {
				logger.Error("计算用户得分失败", zap.Error(err), zap.Int64("summonerID", summonerID))
				return nil
			}
			mu.Lock()
			summonerScores = append(summonerScores, actScore)
			//summonerIDMapScore[summonerID] = *actScore
			mu.Unlock()
			return nil
		})
	}
	scoreCfg := global.GetScoreConf()
	clientCfg := global.GetClientUserConf()
	_ = g.Wait()
	if len(summonerScores) > 0 {
		fmt.Println("敌方用户详情:")
	}
	slices.SortFunc(summonerScores, func(a, b *lcu.UserScore) int {
		return cmp.Compare(b.Score, a.Score)
	})
	// 根据所有用户的分数判断小代上等马中等马下等马
	for _, score := range summonerScores {
		var horse string
		for i, v := range scoreCfg.Horse {
			if score.Score >= v.Score {
				horse = clientCfg.HorseNameConf[i]
				break
			}
		}
		currKDASb := strings.Builder{}
		for i := 0; i < 5 && i < len(score.CurrKDA); i++ {
			currKDASb.WriteString(fmt.Sprintf("%d/%d/%d  ", score.CurrKDA[i][0], score.CurrKDA[i][1],
				score.CurrKDA[i][2]))
		}
		currKDAMsg := currKDASb.String()
		//log.Printf("敌方用户:%s (%s) 得分:%.2f,kda:%s\n", score.SummonerName, horse, score.Score, currKDAMsg)
		fmt.Printf("%s(%d): %s %s\n", horse, int(score.Score), score.SummonerName,
			currKDAMsg)
	}
	allMsg := ""
	// 发送到选人界面
	for _, scoreInfo := range summonerScores {
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
			currKDAMsg, global.Conf.AdaptChatWebsiteTitle)
		allMsg += msg + "\n"
	}
	_ = clipboard.WriteAll(allMsg)
}
func (p *Prophet) onChampSelectSessionUpdate(sessionInfo *models.ChampSelectSessionInfo) error {
	var userPickActionID, userBanActionID, pickChampionID int
	var isSelfPick, isSelfBan, pickIsInProgress, banIsInProgress bool
	alloyPrePickChampionIDSet := make(map[int]struct{}, 5)
	if len(sessionInfo.Actions) == 0 {
		return nil
	}
	for _, actions := range sessionInfo.Actions {
		for _, action := range actions {
			if action.IsAllyAction && action.Type == lcu.ChampSelectPatchTypePick && action.ChampionId > 0 {
				alloyPrePickChampionIDSet[action.ChampionId] = struct{}{}
			}
			if action.ActorCellId != sessionInfo.LocalPlayerCellId {
				continue
			}
			if action.Type == lcu.ChampSelectPatchTypePick {
				isSelfPick = true
				userPickActionID = action.Id
				pickChampionID = action.ChampionId
				pickIsInProgress = action.IsInProgress
			} else if action.Type == lcu.ChampSelectPatchTypeBan {
				isSelfBan = true
				userBanActionID = action.Id
				banIsInProgress = action.IsInProgress
			}
			break
		}
	}
	clientCfg := global.GetClientUserConf()
	if clientCfg.AutoPickChampID > 0 && isSelfPick {
		if pickIsInProgress {
			_ = lcu.PickChampion(clientCfg.AutoPickChampID, userPickActionID)
		} else if pickChampionID == 0 {
			_ = lcu.PrePickChampion(clientCfg.AutoPickChampID, userPickActionID)
		}
	}
	if clientCfg.AutoBanChampID > 0 && isSelfBan && banIsInProgress {
		if _, exist := alloyPrePickChampionIDSet[clientCfg.AutoBanChampID]; !exist {
			_ = lcu.BanChampion(clientCfg.AutoBanChampID, userBanActionID)
		}
	}
	return nil
}
