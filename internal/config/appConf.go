package config

import (
	"github.com/real-web-world/hh-lol-prophet/pkg/lcu/model"
)

const GetRemoteConfApi = "https://lol.buffge.com/api/v1/lol/client/getConf"

// mode
const (
	ModeDebug Mode = "debug"
	ModeProd  Mode = "prod"
)

const (
	DefaultTeamSummonerCount = 5
)

var (
	DefaultClientUserConf = &ClientUserConf{
		AutoAcceptGame:                 false,
		AutoPickChampID:                0,
		AutoBanChampID:                 0,
		AutoSendTeamHorse:              true,
		ShouldSendSelfHorse:            true,
		HorseNameConf:                  [6]string{"通天代", "小代", "上等马", "中等马", "下等马", "牛马"},
		ChooseSendHorseMsg:             [6]bool{true, true, true, true, true, true},
		ChooseChampSendMsgDelaySec:     3,
		ShouldInGameSaveMsgToClipBoard: true,
		ShouldAutoOpenBrowser:          new(false),
	}
	DefaultAppConf = &AppConfig{
		Mode: ModeProd,
		Log: LogConf{
			Level: "info",
		},
		HttpServer: HttpServerConf{
			Addr: "127.0.0.1:4396",
		},
		BuffApi: BuffApi{
			Url:     "https://lol.buffge.com/api",
			Timeout: 5,
		},
		CalcScore: CalcScoreConf{
			Enabled:         true,
			GameMinDuration: 900,
			AllowQueueIDList: []int{
				int(model.RankSoleQueueID),
				int(model.NormalQueueID),
				int(model.RankFlexQueueID),
				int(model.ARAMQueueID),
				int(model.URFQueueID),
				int(model.CheeryQueueID),
				int(model.HexARAMQueueID),
				int(model.ClassicModePvpQueueID),
			},
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
			KillRate: []RateItemConf{
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
			HurtRate: []RateItemConf{
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
			AssistRate: []RateItemConf{
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
			Horse: [6]HorseScoreConf{
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
		AppName:               "lol对局先知",
		WebsiteTitle:          "lol.buffge.com",
		AdaptChatWebsiteTitle: "lol.buffge点康姆",
		ProjectUrl:            "github.com/real-web-world/hh-lol-prophet",
		WebView: WebViewConf{
			IndexUrl: "https://lol.buffge.com/dev/client",
		},
	}
)

type (
	Config struct {
		App    *AppConfig
		Client *ClientUserConf
	}
	AppConfig struct {
		Mode                  Mode           `json:"mode"                  default:"prod"                                     env:"PROPHET_MODE"`
		Log                   LogConf        `json:"log"`
		HttpServer            HttpServerConf `json:"httpServer"`
		BuffApi               BuffApi        `json:"buffApi"                                                                                     required:"true"`
		CalcScore             CalcScoreConf  `json:"calcScore"                                                                                   required:"true"`
		AppName               string         `json:"appName"               default:"lol对局先知"`
		WebsiteTitle          string         `json:"websiteTitle"          default:"lol.buffge.com"`
		AdaptChatWebsiteTitle string         `json:"adaptChatWebsiteTitle" default:"lol.buffge点康姆"`
		ProjectUrl            string         `json:"projectUrl"            default:"github.com/real-web-world/hh-lol-prophet"`
		WebView               WebViewConf    `json:"webView"`
	}
	HttpServerConf struct {
		Addr string `json:"addr" default:"127.0.0.1:4396"`
	}
	WebViewConf struct {
		IndexUrl string `json:"indexUrl" default:"https://lol.buffge.com/dev/client"`
	}
	Mode    = string
	LogConf struct {
		Level string `json:"level" default:"info" env:"logLevel"`
	}
	BuffApi struct {
		Url     string `json:"url"     default:"https://lol.buffge.com/api" env:"buffApiUrl"`
		Timeout int    `json:"timeout" default:"5"`
	}
	RateItemConf struct {
		Limit     float64      `json:"limit"     required:"true"` // >30%
		ScoreConf [][2]float64 `json:"scoreConf" required:"true"` // [ [最低人头限制,加分数] ]
	}
	HorseScoreConf struct {
		Score float64 `json:"score,omitempty" required:"true"`
		Name  string  `json:"name"            required:"true"`
	}
	CalcScoreConf struct {
		Enabled                    bool              `json:"enabled"                    default:"false"`
		GameMinDuration            int               `json:"gameMinDuration"            default:"900"`                   // 允许计算战绩的最低游戏时长
		AllowQueueIDList           []int             `json:"allowQueueIDList"`                                           // 允许计算战绩的queueID
		QueueIDMaTeamSummonerCount map[int]int       `json:"queueIDMaTeamSummonerCount"`                                 // 不同queueID队伍玩家数量 默认5
		FirstBlood                 [2]float64        `json:"firstBlood"                                 required:"true"` // [击杀+,助攻+]
		PentaKills                 [1]float64        `json:"pentaKills"                                 required:"true"` // 五杀
		QuadraKills                [1]float64        `json:"quadraKills"                                required:"true"` // 四杀
		TripleKills                [1]float64        `json:"tripleKills"                                required:"true"` // 三杀
		JoinTeamRateRank           [4]float64        `json:"joinTeamRate"                               required:"true"` // 参团率排名
		GoldEarnedRank             [4]float64        `json:"goldEarned"                                 required:"true"` // 打钱排名
		HurtRank                   [2]float64        `json:"hurtRank"                                   required:"true"` // 伤害排名
		Money2hurtRateRank         [2]float64        `json:"money2HurtRateRank"                         required:"true"` // 金钱转换伤害比排名
		VisionScoreRank            [2]float64        `json:"visionScoreRank"                            required:"true"` // 视野得分排名
		MinionsKilled              [][2]float64      `json:"minionsKilled"                              required:"true"` // 补兵 [ [补兵数,加分数] ]
		KillRate                   []RateItemConf    `json:"killRate"                                   required:"true"` // 人头占比
		HurtRate                   []RateItemConf    `json:"hurtRate"                                   required:"true"` // 伤害占比
		AssistRate                 []RateItemConf    `json:"assistRate"                                 required:"true"` // 助攻占比
		AdjustKDA                  [2]float64        `json:"adjustKDA"                                  required:"true"` // kda
		Horse                      [6]HorseScoreConf `json:"horse"                                      required:"true"` // 马匹名称
		MergeMsg                   bool              `json:"mergeMsg"`                                                   // 是否合并消息为一条
		StrReplaceMap              map[string]string `json:"strReplaceMap"`                                              // 字符串重写 过敏感词系统
	}
)

func (cfg Config) IsDebug() bool {
	return cfg.App.Mode == ModeDebug
}

func (cfg Config) IsProd() bool {
	return !cfg.IsDebug()
}

func (cfg CalcScoreConf) GetTeamSummonerCount(queueID int) int {
	if count, ok := cfg.QueueIDMaTeamSummonerCount[queueID]; ok {
		return count
	}
	return DefaultTeamSummonerCount
}
