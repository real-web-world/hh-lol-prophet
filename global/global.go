package global

import (
	"log"
	"sync"

	"go.uber.org/zap"

	"github.com/real-web-world/hh-lol-prophet/conf"
	"github.com/real-web-world/hh-lol-prophet/pkg/logger"
)

type (
	UserInfo struct {
		IP    string `json:"ip"`
		Mac   string `json:"mac"`
		CpuID string `json:"cpuID"`
	}
)

const (
	LogWriterCleanupKey   = "logWriter"
	sentryDsn             = "https://1c762696e30c4febbb6f8cbcf8835603@o1144230.ingest.sentry.io/6207862"
	buffApiUrl            = "https://lol.buffge.com"
	defaultLogPath        = "./logs/hh-lol-prophet.log"
	WebsiteTitle          = "lol.buffge.com"
	AdaptChatWebsiteTitle = "lol.buffge点康姆"
	AppName               = "lol对局先知"
)

var (
	userInfo    = UserInfo{}
	scoreConfMu = sync.Mutex{}
	Conf        = &conf.AppConf{
		Mode: conf.ModeProd,
		Sentry: conf.SentryConf{
			Enabled: true,
			Dsn:     sentryDsn,
		},
		PProf: conf.PProfConf{
			Enable: true,
		},
		Log: conf.LogConf{
			Level:    logger.LevelInfoStr,
			Filepath: defaultLogPath,
		},
		BuffApi: conf.BuffApi{
			Url:     buffApiUrl,
			Timeout: 3,
		},
		CalcScore: conf.CalcScoreConf{
			Enabled:            true,
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
			Horse: []conf.HorseScoreConf{
				{Score: 180, Name: "通天代"},
				{Score: 150, Name: "小代"},
				{Score: 125, Name: "上等马"},
				{Score: 105, Name: "中等马"},
				{Score: 95, Name: "下等马"},
				{Score: 0.0001, Name: "牛马"},
			},
		},
	}
	Logger   *zap.SugaredLogger
	Cleanups = make(map[string]func() error)
)

func SetUserInfo(info UserInfo) {
	userInfo = info
}
func GetUserInfo() UserInfo {
	return userInfo
}
func Cleanup() {
	for name, cleanup := range Cleanups {
		if err := cleanup(); err != nil {
			log.Printf("%s cleanup err:%v\n", name, err)
		}
	}
	if fn, ok := Cleanups[LogWriterCleanupKey]; ok {
		_ = fn()
	}
}
func IsDevMode() bool {
	return GetEnv() == conf.ModeDebug
}
func GetEnv() conf.Mode {
	return Conf.Mode
}
func GetScoreConf() conf.CalcScoreConf {
	scoreConfMu.Lock()
	defer scoreConfMu.Unlock()
	return Conf.CalcScore
}
func SetScoreConf(scoreConf conf.CalcScoreConf) {
	scoreConfMu.Lock()
	Conf.CalcScore = scoreConf
	scoreConfMu.Unlock()
	return
}
