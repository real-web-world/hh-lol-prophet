package conf

const (
	ModeDebug Mode = "debug"
	ModeProd  Mode = "prod"
)

type (
	AppConf struct {
		Mode                  Mode          `json:"mode" default:"prod" env:"mode"`
		PProf                 PProfConf     `json:"pprof"`
		BuffApi               BuffApi       `json:"buffApi" required:"true"`
		CalcScore             CalcScoreConf `json:"calcScore" required:"true"`
		AppName               string        `json:"appName" default:"lol对局先知"`
		WebsiteTitle          string        `json:"websiteTitle" default:"lol.buffge.com"`
		AdaptChatWebsiteTitle string        `json:"adaptChatWebsiteTitle" default:"lol.buffge点康姆"`
		ProjectUrl            string        `json:"projectUrl" default:"github.com/real-web-world/hh-lol-prophet"`
		Otlp                  OtlpConf      `json:"otlp"`
		WebView               WebViewConf   `json:"webView"`
	}
	WebViewConf struct {
		IndexUrl string `json:"indexUrl" default:"https://lol.buffge.com/dev/client"`
	}
	Mode      = string
	PProfConf struct {
		Enable bool `default:"false" env:"enablePProf" json:"enable"`
	}
	LogConf struct {
		Level string `json:"level" default:"info" env:"logLevel"`
	}
	OtlpConf struct {
		EndpointUrl string `json:"endpointUrl" default:"https://otlp-gateway-prod-ap-southeast-1.grafana.net/otlp"`
		Token       string `json:"token" default:"ODE5OTIyOmdsY19leUp2SWpvaU16QXdOekkzSWl3aWJpSTZJbk4wWVdOckxUZ3hPVGt5TWkxdmRHeHdMWGR5YVhSbExXOTBiSEF0ZEc5clpXNHRNaUlzSW1zaU9pSTVORVl5TVdsS1pHdG9NVmN3VXpaaE1HczNhakZwYm1jaUxDSnRJanA3SW5JaU9pSndjbTlrTFdGd0xYTnZkWFJvWldGemRDMHhJbjE5"`
	}
	BuffApi struct {
		Url     string `json:"url" default:"https://k2-api.buffge.com:40012/prod/lol" env:"buffApiUrl"`
		Timeout int    `json:"timeout" default:"5"`
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
		Horse              [6]HorseScoreConf `json:"horse" required:"true"`              // 马匹名称
		MergeMsg           bool              `json:"mergeMsg"`                           // 是否合并消息为一条
	}
)
