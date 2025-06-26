package global

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/real-web-world/hh-lol-prophet/conf"
	"github.com/real-web-world/hh-lol-prophet/services/lcu/models"
)

type (
	AppInfo struct {
		Version   string
		Commit    string
		BuildUser string
		BuildTime string
	}
	UserInfo struct {
		MacHash  string
		Summoner *models.SummonerProfileData
	}
)

// envKey
const (
	EnvKeyMode = "PROPHET_MODE"
)

const (
	ZapLoggerCleanupKey = "ZapLogger"
	LogWriterCleanupKey = "logWriter"
	OtelCleanupKey      = "otel"
)

var (
	cleanupsMu                      = &sync.Mutex{}
	defaultShouldAutoOpenBrowserCfg = true
	DefaultClientUserConf           = conf.ClientUserConf{
		AutoAcceptGame:                 false,
		AutoPickChampID:                0,
		AutoBanChampID:                 0,
		AutoSendTeamHorse:              true,
		ShouldSendSelfHorse:            true,
		HorseNameConf:                  [6]string{"通天代", "小代", "上等马", "中等马", "下等马", "牛马"},
		ChooseSendHorseMsg:             [6]bool{true, true, true, true, true, true},
		ChooseChampSendMsgDelaySec:     3,
		ShouldInGameSaveMsgToClipBoard: true,
		ShouldAutoOpenBrowser:          &defaultShouldAutoOpenBrowserCfg,
	}
	DefaultAppConf = conf.AppConf{
		CalcScore: conf.CalcScoreConf{
			Enabled:            true,
			GameMinDuration:    900,
			AllowQueueIDList:   []int{430, 420, 450, 440, 1700},
			FirstBlood:         [2]float64{10, 5},
			PentaKills:         [1]float64{20},
			QuadraKills:        [1]float64{10},
			TripleKills:        [1]float64{5},
			JoinTeamRateRank:   [4]float64{10, 5, 5, 10},
			GoldEarnedRank:     [4]float64{10, 5, 5, 10},
			HurtRank:           [2]float64{10, 5},
			Money2hurtRateRank: [2]float64{10, 5},
			VisionScoreRank:    [2]float64{10, 5},
			MinionsKilled: [][2]float64{
				{10, 20},
				{9, 10},
				{8, 5},
			},
			KillRate: []conf.RateItemConf{
				{Limit: 50, ScoreConf: [][2]float64{
					{15, 40},
					{10, 20},
					{5, 10},
				}},
				{Limit: 40, ScoreConf: [][2]float64{
					{15, 20},
					{10, 10},
					{5, 5},
				}},
			},
			HurtRate: []conf.RateItemConf{
				{Limit: 40, ScoreConf: [][2]float64{
					{15, 40},
					{10, 20},
					{5, 10},
				}},
				{Limit: 30, ScoreConf: [][2]float64{
					{15, 20},
					{10, 10},
					{5, 5},
				}},
			},
			AssistRate: []conf.RateItemConf{
				{Limit: 50, ScoreConf: [][2]float64{
					{20, 30},
					{18, 25},
					{15, 20},
					{10, 10},
					{5, 5},
				}},
				{Limit: 40, ScoreConf: [][2]float64{
					{20, 15},
					{15, 10},
					{10, 5},
					{5, 3},
				}},
			},
			AdjustKDA: [2]float64{2, 5},
			Horse: [6]conf.HorseScoreConf{
				{Score: 180, Name: "通天代"},
				{Score: 150, Name: "小代"},
				{Score: 125, Name: "上等马"},
				{Score: 105, Name: "中等马"},
				{Score: 95, Name: "下等马"},
				{Score: 0.0001, Name: "牛马"},
			},
			MergeMsg: false,
			StrReplaceMap: map[string]string{
				"0": "𝟘",
				"1": "𝟙",
				"2": "𝟚",
				"3": "𝟛",
				"4": "𝟜",
				"5": "𝟝",
				"6": "𝟞",
				"7": "𝟟",
				"8": "𝟠",
				"9": "𝟡",
				"马": "⾺",
			},
		},
	}
	userInfo       = &UserInfo{}
	confMu         = sync.Mutex{}
	Conf           = new(conf.AppConf)
	ClientUserConf = new(conf.ClientUserConf)
	Logger         *zap.SugaredLogger
	Cleanups       = make(map[string]func(c context.Context) error)
	AppBuildInfo   = AppInfo{}
)

// DB
var (
	SqliteDB *gorm.DB
)

func SetUserMac(userMacHash string) {
	confMu.Lock()
	userInfo.MacHash = userMacHash
	confMu.Unlock()
}
func SetCurrSummoner(summoner *models.SummonerProfileData) {
	confMu.Lock()
	userInfo.Summoner = summoner
	confMu.Unlock()
}
func GetUserInfo() UserInfo {
	confMu.Lock()
	defer confMu.Unlock()
	return *userInfo
}
func Cleanup() {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()
	for name, cleanup := range Cleanups {
		if name == LogWriterCleanupKey {
			continue
		}
		if err := cleanup(ctx); err != nil {
			log.Printf("%s cleanup err:%v\n", name, err)
		}
	}
	if fn, ok := Cleanups[LogWriterCleanupKey]; ok {
		_ = fn(ctx)
	}
}
func IsDevMode() bool {
	return GetEnv() == conf.ModeDebug
}
func IsProdMode() bool {
	return !IsDevMode()
}
func GetEnv() conf.Mode {
	return Conf.Mode
}

func GetEnvMode() conf.Mode {
	return os.Getenv(EnvKeyMode)
}
func IsEnvModeDev() bool {
	return GetEnvMode() == conf.ModeDebug
}

func GetScoreConf() conf.CalcScoreConf {
	confMu.Lock()
	defer confMu.Unlock()
	return Conf.CalcScore
}
func SetScoreConf(scoreConf conf.CalcScoreConf) {
	confMu.Lock()
	Conf.CalcScore = scoreConf
	confMu.Unlock()
	return
}
func GetClientUserConf() conf.ClientUserConf {
	confMu.Lock()
	defer confMu.Unlock()
	data := *ClientUserConf
	return data
}
func SetClientUserConf(cfg conf.UpdateClientUserConfReq) *conf.ClientUserConf {
	confMu.Lock()
	defer confMu.Unlock()
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
	return ClientUserConf
}
func SetAppInfo(info AppInfo) {
	AppBuildInfo = info
}
func SetCleanup(name string, fn func(c context.Context) error) {
	cleanupsMu.Lock()
	Cleanups[name] = fn
	cleanupsMu.Unlock()
}
