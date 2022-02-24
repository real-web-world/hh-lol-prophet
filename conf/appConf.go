package conf

import (
	"github.com/real-web-world/hh-lol-prophet/pkg/logger"
)

const (
	ModeDebug Mode = "debug"
	ModeProd  Mode = "prod"
)

type (
	Mode    = string
	AppConf struct {
		Mode      Mode          `json:"mode" default:"prod" env:"mode"`
		Sentry    SentryConf    `json:"sentry"`
		PProf     PProfConf     `json:"pprof"`
		Log       LogConf       `json:"log" required:"true"`
		BuffApi   BuffApi       `json:"buffApi" required:"true"`
		CalcScore CalcScoreConf `json:"calcScore" required:"true"`
	}
	SentryConf struct {
		Enabled bool   `json:"enabled" default:"false" env:"enableSentry"`
		Dsn     string `json:"dsn"`
	}
	PProfConf struct {
		Enable bool `default:"false" env:"enablePProf" json:"enable"`
	}
	LogConf struct {
		Level      logger.LogLevelStr `json:"level" default:"info" env:"logLevel"`
		Filepath   string             `required:"true" json:"filepath" env:"logFilepath"`
		MaxSize    int                `default:"1024" env:"logMaxSize"`
		MaxBackups int                `default:"7" env:"logMaxBackups"`
		MaxAge     int                `default:"7" env:"logMaxAge"`
		Compress   bool               `default:"true" env:"logCompress"`
	}
	BuffApi struct {
		Url     string `json:"url"`
		Timeout int    `json:"timeout"`
	}
	RateItemConf struct {
		Limit     float64      `json:"limit" required:"true"`     // >30%
		ScoreConf [][2]float64 `json:"scoreConf" required:"true"` // [ [最低人头限制,加分数] ]
	}
	HorseScoreConf struct {
		Score float64 `json:"score,omitempty" required:"true"`
		Name  string  `json:"name" required:"true"`
	}
	CalcScoreConf struct {
		Enabled            bool              `json:"enabled" default:"false"`
		FirstBlood         [2]float64        `json:"firstBlood" required:"true"`         // [击杀+,助攻+]
		PentaKills         [1]float64        `json:"pentaKills" required:"true"`         // 五杀
		QuadraKills        [1]float64        `json:"quadraKills" required:"true"`        // 四杀
		TripleKills        [1]float64        `json:"tripleKills" required:"true"`        // 三杀
		JoinTeamRateRank   [4]float64        `json:"joinTeamRate" required:"true"`       // 参团率排名
		GoldEarnedRank     [4]float64        `json:"goldEarned" required:"true"`         // 打钱排名
		HurtRank           [2]float64        `json:"hurtRank" required:"true"`           // 伤害排名
		Money2hurtRateRank [2]float64        `json:"money2HurtRateRank" required:"true"` // 金钱转换伤害比排名
		VisionScoreRank    [2]float64        `json:"visionScoreRank" required:"true"`    // 视野得分排名
		MinionsKilled      [][2]float64      `json:"minionsKilled" required:"true"`      // 补兵 [ [补兵数,加分数] ]
		KillRate           []RateItemConf    `json:"killRate" required:"true"`           // 人头占比
		HurtRate           []RateItemConf    `json:"hurtRate" required:"true"`           // 伤害占比
		AssistRate         []RateItemConf    `json:"assistRate" required:"true"`         // 助攻占比
		AdjustKDA          [2]float64        `json:"adjustKDA" required:"true"`          // kda
		Horse              [6]HorseScoreConf `json:"horse" required:"true"`
		MergeMsg           bool              `json:"mergeMsg"` // 是否合并消息为一条
	}
)
