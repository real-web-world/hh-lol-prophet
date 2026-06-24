package prophet

import (
	"cmp"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
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

	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"

	appProj "github.com/real-web-world/hh-lol-prophet"
	"github.com/real-web-world/hh-lol-prophet/internal/config"
	"github.com/real-web-world/hh-lol-prophet/pkg/bdk"
	"github.com/real-web-world/hh-lol-prophet/pkg/lcu"
	"github.com/real-web-world/hh-lol-prophet/pkg/lcu/model"
)

type (
	GameState string
	Prophet   struct {
		ctx           context.Context
		opts          *options
		httpSrv       *http.Server
		lcuPort       int
		lcuToken      string
		lcuActive     bool
		lcu           *lcu.Client
		currSummoner  *model.SummonerProfileData
		cancel        func()
		api           *Api
		mu            *sync.Mutex
		confMu        *sync.Mutex
		GameState     GameState
		lcuRP         *lcu.RP
		l             *slog.Logger
		appCfg        *config.AppConfig
		calcScoreCfg  *config.CalcScoreConf
		clientUserCfg *config.ClientUserConf
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

var defaultOpts = &options{
	debug:       false,
	enablePprof: true,
	httpAddr:    "127.0.0.1:4396",
}

var allowOriginRegex = regexp.MustCompile(`.+?\\.buffge\\.com(:\\d+)?$`)

func NewProphet(opts ...ApplyOption) *Prophet {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Prophet{
		ctx:       ctx,
		cancel:    cancel,
		mu:        &sync.Mutex{},
		opts:      defaultOpts,
		GameState: GameStateNone,
	}
	p.api = &Api{p: p}
	for _, fn := range opts {
		fn(p.opts)
	}
	return p
}

func (p *Prophet) GetScoreConf() config.CalcScoreConf {
	p.confMu.Lock()
	defer p.confMu.Unlock()
	return *p.calcScoreCfg
}

func (p *Prophet) SetScoreConf(scoreConf config.CalcScoreConf) {
	p.confMu.Lock()
	*p.calcScoreCfg = scoreConf
	p.confMu.Unlock()
}

func (p *Prophet) GetClientUserConf() config.ClientUserConf {
	p.confMu.Lock()
	defer p.confMu.Unlock()
	data := *p.clientUserCfg
	return data
}

func (p *Prophet) SetClientUserConf(cfg config.UpdateClientUserConfReq) (*config.ClientUserConf, error) {
	p.confMu.Lock()
	defer p.confMu.Unlock()
	ClientUserConf := p.clientUserCfg
	if cfg.AutoAcceptGame != nil {
		ClientUserConf.AutoAcceptGame = *cfg.AutoAcceptGame
	}
	if cfg.AutoPickChampID != nil {
		ClientUserConf.AutoPickChampID = *cfg.AutoPickChampID
	}
	if cfg.AutoBanChampID != nil {
		ClientUserConf.AutoBanChampID = *cfg.AutoBanChampID
	}
	if cfg.AutoSendTeamHorse != nil {
		ClientUserConf.AutoSendTeamHorse = *cfg.AutoSendTeamHorse
	}
	if cfg.ShouldSendSelfHorse != nil {
		ClientUserConf.ShouldSendSelfHorse = *cfg.ShouldSendSelfHorse
	}
	if cfg.HorseNameConf != nil {
		ClientUserConf.HorseNameConf = *cfg.HorseNameConf
	}
	if cfg.ChooseSendHorseMsg != nil {
		ClientUserConf.ChooseSendHorseMsg = *cfg.ChooseSendHorseMsg
	}
	if cfg.ChooseChampSendMsgDelaySec != nil {
		ClientUserConf.ChooseChampSendMsgDelaySec = *cfg.ChooseChampSendMsgDelaySec
	}
	if cfg.ShouldInGameSaveMsgToClipBoard != nil {
		ClientUserConf.ShouldInGameSaveMsgToClipBoard = *cfg.ShouldInGameSaveMsgToClipBoard
	}
	if cfg.ShouldAutoOpenBrowser != nil {
		ClientUserConf.ShouldAutoOpenBrowser = cfg.ShouldAutoOpenBrowser
	}
	return ClientUserConf, nil
}

func (p *Prophet) Run() error {
	go p.MonitorStart()
	go p.captureStartMessage()
	// p.initGin()
	go p.initWebView()
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
				if !errors.Is(err, lcu.ErrLolProcessNotFound) {
					slog.Warn("获取lcu info 失败", slog.Any("err", err))
				}
				time.Sleep(time.Second)
				continue
			}
			p.initLcuClient(port, token)
			err = p.initLcuRP(port, token)
			if err != nil {
				slog.Debug("初始化lcuRP失败", slog.Any("err", err))
			}
			err = p.initGameFlowMonitor(port, token)
			if err != nil {
				slog.Debug("游戏流程监视器 err:", slog.Any("err", err))
			}
			p.SetCurrSummoner(nil)
			p.lcuActive = false
			p.currSummoner = nil
		}
		time.Sleep(time.Second)
	}
}

func (p *Prophet) SetCurrSummoner(summoner *model.SummonerProfileData) {
	p.confMu.Lock()
	p.currSummoner = summoner
	p.confMu.Unlock()
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
	p.lcu = lcu.NewClient(port, token, p.l)
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
	slog.Debug(fmt.Sprintf("connect to lcu %s", u.String()))
	defer func() {
		_ = c.Close()
	}()
	err = bdk.Retry(func() error {
		currSummoner, err := p.lcu.GetSummonerProfile()
		if err == nil {
			p.currSummoner = currSummoner
		}
		return err
	}, time.Second, 5)
	if err != nil {
		return errors.New("获取当前召唤师信息失败:" + err.Error())
	}
	p.SetCurrSummoner(p.currSummoner)
	p.lcuActive = true
	_ = c.WriteMessage(websocket.TextMessage, lcu.SubscribeAllEventMsg)
	for {
		msgType, message, err := c.ReadMessage()
		if err != nil {
			slog.Debug("lol事件监控读取消息失败", slog.Any("err", err))
			return err
		}
		msg := &lcu.WsMsg{}
		if msgType != websocket.TextMessage || len(message) < lcu.OnJsonApiEventPrefixLen+1 {
			continue
		}
		_ = json.Unmarshal(message[lcu.OnJsonApiEventPrefixLen:len(message)-1], msg)
		switch msg.Uri {
		case string(lcu.WsEvtGameFlowChanged):
			gameFlow := new(model.GameFlow)
			_ = json.Unmarshal(msg.Data, gameFlow)
			p.onGameFlowUpdate(*gameFlow)
		case string(lcu.WsEvtChampSelectUpdateSession):
			sessionInfo := &model.ChampSelectSessionInfo{}
			if err = json.Unmarshal(msg.Data, sessionInfo); err != nil {
				slog.Debug("champSelectUpdateSessionEvt 解析结构体失败", slog.Any("err", err))
				continue
			}
			go func() {
				_ = p.onChampSelectSessionUpdate(sessionInfo)
			}()
		default:

		}
	}
}

func (p *Prophet) onGameFlowUpdate(gameFlow model.GameFlow) {
	slog.Debug("切换状态:" + string(gameFlow))
	switch gameFlow {
	case model.GameFlowChampionSelect:
		slog.Info("进入英雄选择阶段,正在计算用户分数")
		p.updateGameState(GameStateChampSelect)
		go p.ChampionSelectStart()
	case model.GameFlowNone:
		p.updateGameState(GameStateNone)
	case model.GameFlowMatchmaking:
		p.updateGameState(GameStateMatchmaking)
	case model.GameFlowInProgress:
		p.updateGameState(GameStateInGame)
		go p.CalcEnemyTeamScore()
	case model.GameFlowReadyCheck:
		p.updateGameState(GameStateReadyCheck)
		clientCfg := p.GetClientUserConf()
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
	slog.Info(p.appCfg.AppName + "已启动")
}

func (p *Prophet) initWebView() {
	clientCfg := p.GetClientUserConf()
	indexUrl := p.appCfg.WebView.IndexUrl
	defaultUrl := indexUrl + "?version=" + appProj.APPVersion
	websiteUrl := defaultUrl
	if clientCfg.ShouldAutoOpenBrowser != nil && !*clientCfg.ShouldAutoOpenBrowser {
		log.Println("自动打开浏览器选项已关闭,手动打开请访问 " + websiteUrl)
		return
	}
	cmd := exec.Command("cmd", "/c", "start", websiteUrl)
	_ = cmd.Run()
	log.Println("界面已在浏览器中打开,若未打开请手动访问 " + websiteUrl)
}

func (p *Prophet) ChampionSelectStart() {
	clientCfg := p.GetClientUserConf()
	sendConversationMsgDelayCtx, cancel := context.WithTimeout(context.Background(),
		time.Second*time.Duration(clientCfg.ChooseChampSendMsgDelaySec))
	defer cancel()
	var conversationID string
	var summonerIDList []int64
	for i := 0; i < 3; i++ {
		time.Sleep(time.Second)
		// 获取队伍所有用户信息
		conversationID, summonerIDList, _ = p.getTeamUsers()
		if len(summonerIDList) != 5 {
			continue
		}
	}
	//if global.IsDevMode() {
	//	summonerIDList = []int64{2964390005, 4103784618, 4132401993, 4118593599, 4019221688}
	//	// summonerIDList = []int64{4006944917}
	//}
	if len(summonerIDList) == 0 {
		return
	}
	slog.Debug("队伍人员列表:", slog.Any("summonerIDList", summonerIDList))
	// 查询所有用户的信息并计算得分
	g := errgroup.Group{}
	summonerScores := make([]*lcu.UserScore, 0, 5)
	mu := sync.Mutex{}
	summonerIDMapInfo, err := p.listSummoner(summonerIDList)
	if err != nil {
		slog.Error("查询召唤师信息失败", slog.Any("err", err), slog.Any("summonerIDList", summonerIDList))
		return
	}
	for _, summoner := range summonerIDMapInfo {
		summoner := summoner
		summonerID := summoner.SummonerId
		g.Go(func() error {
			actScore, err := p.GetUserScore(summoner)
			if err != nil {
				slog.Error("计算用户得分失败", slog.Any("err", err), slog.Int64("summonerID", summonerID))
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
	scoreCfg := p.GetScoreConf()
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
		if len(scoreInfo.CurrKDA) == 0 {
			horse = "未查询到战绩"
		}
		currKDASb := strings.Builder{}
		for i := 0; i < 5 && i < len(scoreInfo.CurrKDA); i++ {
			fmt.Fprintf(&currKDASb, "%d-%d-%d  ", scoreInfo.CurrKDA[i][0], scoreInfo.CurrKDA[i][1],
				scoreInfo.CurrKDA[i][2])
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
		_ = p.SendConversationMsg(msg, conversationID)
		time.Sleep(time.Millisecond * 2100)
	}
	if !clientCfg.AutoSendTeamHorse {
		_ = bdk.WriteClipboard(allMsg)
		fmt.Println("已将队伍马匹信息复制到剪切板 ", time.Now().Format(time.DateTime))
		fmt.Println()
		fmt.Println(allMsg)
		return
	}
	if scoreCfg.MergeMsg {
		_ = p.SendConversationMsg(mergedMsg, conversationID)
	}
}

func (p *Prophet) AcceptGame() {
	_ = p.lcu.AcceptGame()
}

func (p *Prophet) CalcEnemyTeamScore() {
	// 获取当前游戏进程
	session, err := p.lcu.QueryGameFlowSession()
	if err != nil {
		return
	}
	if session.Phase != model.GameFlowInProgress {
		return
	}
	if p.currSummoner == nil {
		return
	}
	selfID := p.currSummoner.SummonerId
	selfTeamUsers, enemyTeamUsers := p.getAllUsersFromSession(selfID, session)
	_ = selfTeamUsers
	summonerIDList := enemyTeamUsers
	// if !false && global.IsDevMode() {
	// 	summonerIDList = []int64{2964390005, 4103784618, 4132401993, 4118593599, 4019221688}
	// 	// summonerIDList = []int64{4006944917}
	// }
	slog.Debug("敌方队伍人员列表:", slog.Any("summonerIDList", summonerIDList))
	if len(summonerIDList) == 0 {
		return
	}
	// 查询所有用户的信息并计算得分
	g := errgroup.Group{}
	summonerScores := make([]*lcu.UserScore, 0, 5)
	mu := sync.Mutex{}
	summonerIDMapInfo, err := p.listSummoner(summonerIDList)
	if err != nil {
		slog.Error("查询召唤师信息失败", slog.Any("err", err), slog.Any("summonerIDList", summonerIDList))
		return
	}
	for _, summoner := range summonerIDMapInfo {
		summoner := summoner
		summonerID := summoner.SummonerId
		g.Go(func() error {
			actScore, err := p.GetUserScore(summoner)
			if err != nil {
				slog.Error("计算用户得分失败", slog.Any("err", err), slog.Int64("summonerID", summonerID))
				return nil
			}
			mu.Lock()
			summonerScores = append(summonerScores, actScore)
			// summonerIDMapScore[summonerID] = *actScore
			mu.Unlock()
			return nil
		})
	}
	scoreCfg := p.GetScoreConf()
	clientCfg := p.GetClientUserConf()
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
			fmt.Fprintf(&currKDASb, "%d-%d-%d  ", score.CurrKDA[i][0], score.CurrKDA[i][1],
				score.CurrKDA[i][2])
		}
		currKDAMsg := currKDASb.String()
		// log.Printf("敌方用户:%s (%s) 得分:%.2f,kda:%s\n", score.SummonerName, horse, score.Score, currKDAMsg)
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
			fmt.Fprintf(&currKDASb, "%d-%d-%d  ", scoreInfo.CurrKDA[i][0], scoreInfo.CurrKDA[i][1],
				scoreInfo.CurrKDA[i][2])
		}
		currKDAMsg := currKDASb.String()
		if len(currKDAMsg) > 0 {
			currKDAMsg = currKDAMsg[:len(currKDAMsg)-1]
		}
		msg := fmt.Sprintf("%s(%d): %s %s  -- %s", horse, int(scoreInfo.Score), scoreInfo.SummonerName,
			currKDAMsg, p.appCfg.AdaptChatWebsiteTitle)
		allMsg += msg + "\n"
	}
	_ = bdk.WriteClipboard(allMsg)
}

func (p *Prophet) onChampSelectSessionUpdate(sessionInfo *model.ChampSelectSessionInfo) error {
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
			switch action.Type {
			case lcu.ChampSelectPatchTypePick:
				isSelfPick = true
				userPickActionID = action.Id
				pickChampionID = action.ChampionId
				pickIsInProgress = action.IsInProgress
			case lcu.ChampSelectPatchTypeBan:
				isSelfBan = true
				userBanActionID = action.Id
				banIsInProgress = action.IsInProgress
			}
			break
		}
	}
	clientCfg := p.GetClientUserConf()
	if clientCfg.AutoPickChampID != 0 && isSelfPick {
		if pickIsInProgress {
			_ = p.lcu.PickChampion(clientCfg.AutoPickChampID, userPickActionID)
		} else if pickChampionID == 0 {
			_ = p.lcu.PrePickChampion(clientCfg.AutoPickChampID, userPickActionID)
		}
	}
	if clientCfg.AutoBanChampID != 0 && isSelfBan && banIsInProgress {
		if _, exist := alloyPrePickChampionIDSet[clientCfg.AutoBanChampID]; !exist {
			_ = p.lcu.BanChampion(clientCfg.AutoBanChampID, userBanActionID)
		}
	}
	return nil
}
