package lcu

import (
	"fmt"
	"strings"
)

type (
	UserScore struct {
		SummonerID   int64    `json:"summonerID"`
		SummonerName string   `json:"summonerName"`
		Score        float64  `json:"score"`
		CurrKDA      [][3]int `json:"currKDA"`
	}
	IncScoreReason struct {
		reason ScoreOption
		incVal float64
	}
	ScoreWithReason struct {
		score   float64
		reasons []IncScoreReason
	}
	ScoreOption string // 得分选项
)

const (
	ScoreOptionFirstBloodKill     ScoreOption = "一血击杀"
	ScoreOptionFirstBloodAssist   ScoreOption = "一血助攻"
	ScoreOptionPentaKills         ScoreOption = "五杀"
	ScoreOptionQuadraKills        ScoreOption = "四杀"
	ScoreOptionTripleKills        ScoreOption = "三杀"
	ScoreOptionJoinTeamRateRank   ScoreOption = "参团率排名"
	ScoreOptionGoldEarnedRank     ScoreOption = "打钱排名"
	ScoreOptionHurtRank           ScoreOption = "伤害排名"
	ScoreOptionMoney2hurtRateRank ScoreOption = "金钱转换伤害比排名"
	ScoreOptionVisionScoreRank    ScoreOption = "视野得分排名"
	ScoreOptionMinionsKilled      ScoreOption = "补兵"
	ScoreOptionKillRate           ScoreOption = "击杀占比"
	ScoreOptionHurtRate           ScoreOption = "伤害占比"
	ScoreOptionAssistRate         ScoreOption = "助攻占比"
	ScoreOptionKDAAdjust          ScoreOption = "kda微调"
)

func NewScoreWithReason(score float64) *ScoreWithReason {
	return &ScoreWithReason{
		score:   score,
		reasons: make([]IncScoreReason, 0, 5),
	}
}
func (s *ScoreWithReason) Add(incVal float64, reason ScoreOption) {
	s.score += incVal
	s.reasons = append(s.reasons, IncScoreReason{
		reason: reason,
		incVal: incVal,
	})
}
func (s *ScoreWithReason) Value() float64 {
	return s.score
}
func (s *ScoreWithReason) Reasons2String() string {
	sb := strings.Builder{}
	for _, reason := range s.reasons {
		sb.WriteString(fmt.Sprintf("%s%.2f,", reason.reason, reason.incVal))
	}
	return sb.String()
}
