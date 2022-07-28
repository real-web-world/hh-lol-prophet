package hh_lol_prophet

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/real-web-world/hh-lol-prophet/global"
	"github.com/real-web-world/hh-lol-prophet/services/lcu"
	"github.com/real-web-world/hh-lol-prophet/services/lcu/models"

	"github.com/real-web-world/hh-lol-prophet/pkg/bdk"
	"github.com/real-web-world/hh-lol-prophet/services/logger"
)

const (
	defaultScore       = 100 // 默认分数
	minGameDurationSec = 15 * 60
)

var (
	SendConversationMsg   = lcu.SendConversationMsg
	ListConversationMsg   = lcu.ListConversationMsg
	GetCurrConversationID = lcu.GetCurrConversationID
	QuerySummoner         = lcu.QuerySummoner
	QueryGameSummary      = lcu.QueryGameSummary
	ListGamesBySummonerID = lcu.ListGamesBySummonerID
)

func getTeamUsers() (string, []int64, error) {
	conversationID, err := GetCurrConversationID()
	if err != nil {
		return "", nil, err
	}
	msgList, err := ListConversationMsg(conversationID)
	if err != nil {
		return "", nil, err
	}
	summonerIDList := getSummonerIDListFromConversationMsgList(msgList)
	return conversationID, summonerIDList, nil
}
func getSummonerIDListFromConversationMsgList(msgList []lcu.ConversationMsg) []int64 {
	summonerIDList := make([]int64, 0, 5)
	for _, msg := range msgList {
		if msg.Type == lcu.ConversationMsgTypeSystem && msg.Body == lcu.JoinedRoomMsg {
			summonerIDList = append(summonerIDList, msg.FromSummonerId)
		}
	}
	return summonerIDList
}

func GetUserScore(summonerID int64) (*lcu.UserScore, error) {
	userScoreInfo := &lcu.UserScore{
		SummonerID: summonerID,
		Score:      defaultScore,
	}
	// 获取用户信息
	summoner, err := QuerySummoner(summonerID)
	if err != nil {
		return nil, err
	}
	userScoreInfo.SummonerName = summoner.DisplayName
	// 获取战绩列表
	gameList, err := listGameHistory(summonerID)
	if err != nil {
		logger.Error("获取用户战绩失败", zap.Error(err), zap.Int64("id", summonerID))
		return userScoreInfo, nil
	}
	// 获取每一局战绩
	g := errgroup.Group{}
	gameSummaryList := make([]lcu.GameSummary, 0, len(gameList))
	mu := sync.Mutex{}
	currKDAList := make([][3]int, len(gameList))
	for i, info := range gameList {
		info := info
		currKDAList[len(gameList)-i-1] = [3]int{
			info.Participants[0].Stats.Kills,
			info.Participants[0].Stats.Deaths,
			info.Participants[0].Stats.Assists,
		}
		g.Go(func() error {
			var gameSummary *lcu.GameSummary
			err = retry.Do(func() error {
				var tmpErr error
				gameSummary, tmpErr = QueryGameSummary(info.GameId)
				return tmpErr
			}, retry.Delay(time.Millisecond*10), retry.Attempts(5))
			if err != nil {
				sentry.WithScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelError)
					scope.SetExtra("info", info)
					scope.SetExtra("gameID", info.GameId)
					scope.SetExtra("error", err.Error())
					scope.SetExtra("errorVerbose", errors.Errorf("%+v", err))
					sentry.CaptureMessage("获取游戏对局详细信息失败")
				})
				logger.Debug("获取游戏对局详细信息失败", zap.Error(err), zap.Int64("id", info.GameId))
				return nil
			}
			mu.Lock()
			gameSummaryList = append(gameSummaryList, *gameSummary)
			mu.Unlock()
			return nil
		})
	}
	userScoreInfo.CurrKDA = currKDAList
	err = g.Wait()
	if err != nil {
		logger.Error("获取用户详细战绩失败", zap.Error(err), zap.Int64("id", summonerID))
		return userScoreInfo, nil
	}
	// 分析每一局战绩计算得分
	var totalScore float64 = 0
	totalGameCount := 0
	type gameScoreWithWeight struct {
		score       float64
		isCurrTimes bool
	}
	// gameWeightScoreList := make([]gameScoreWithWeight, 0, len(gameSummaryList))
	nowTime := time.Now()
	currTimeScoreList := make([]float64, 0, 10)
	otherGameScoreList := make([]float64, 0, 10)
	for _, gameSummary := range gameSummaryList {
		gameScore, err := calcUserGameScore(summonerID, gameSummary)
		if err != nil {
			logger.Debug("游戏战绩计算用户得分失败", zap.Error(err), zap.Int64("summonerID", summonerID),
				zap.Int64("gameID", gameSummary.GameId))
			return userScoreInfo, nil
		}
		weightScoreItem := gameScoreWithWeight{
			score:       gameScore.Value(),
			isCurrTimes: nowTime.Before(gameSummary.GameCreationDate.Add(time.Hour * 5)),
		}
		if weightScoreItem.isCurrTimes {
			currTimeScoreList = append(currTimeScoreList, gameScore.Value())
		} else {
			otherGameScoreList = append(otherGameScoreList, gameScore.Value())
		}
		totalGameCount++
		totalScore += gameScore.Value()
		// log.Printf("game: %d,得分: %.2f\n", gameSummary.GameId, gameScore)
	}
	totalGameScore := 0.0
	totalTimeScore := 0.0
	avgTimeScore := 0.0
	totalOtherGameScore := 0.0
	avgOtherGameScore := 0.0
	for _, score := range currTimeScoreList {
		totalTimeScore += score
		totalGameScore += score
	}
	for _, score := range otherGameScoreList {
		totalOtherGameScore += score
		totalGameScore += score
	}
	if totalTimeScore > 0 {
		avgTimeScore = totalTimeScore / float64(len(currTimeScoreList))
	}
	if totalOtherGameScore > 0 {
		avgOtherGameScore = totalOtherGameScore / float64(len(otherGameScoreList))
	}
	totalGameAvgScore := 0.0
	if totalGameCount > 0 {
		totalGameAvgScore = totalGameScore / float64(totalGameCount)
	}
	weightTotalScore := 0.0
	// curr time
	{
		if len(currTimeScoreList) == 0 {
			weightTotalScore += .8 * totalGameAvgScore
		} else {
			weightTotalScore += .8 * avgTimeScore
		}
	}
	// other games
	{
		if len(otherGameScoreList) == 0 {
			weightTotalScore += .2 * totalGameAvgScore
		} else {
			weightTotalScore += .2 * avgOtherGameScore
		}
	}
	// 计算平均值返回
	// userScoreInfo.Score = totalScore / float64(totalGameCount)
	if len(gameSummaryList) == 0 {
		weightTotalScore = defaultScore
	}
	userScoreInfo.Score = weightTotalScore
	return userScoreInfo, nil
}

func listGameHistory(summonerID int64) ([]lcu.GameInfo, error) {
	fmtList := make([]lcu.GameInfo, 0, 20)
	resp, err := ListGamesBySummonerID(summonerID, 0, 20)
	if err != nil {
		logger.Error("查询用户战绩失败", zap.Error(err), zap.Int64("summonerID", summonerID))
		return nil, err
	}
	for _, gameItem := range resp.Games.Games {
		if gameItem.QueueId != models.NormalQueueID &&
			gameItem.QueueId != models.RankSoleQueueID &&
			gameItem.QueueId != models.ARAMQueueID &&
			gameItem.QueueId != models.RankFlexQueueID {
			continue
		}
		if gameItem.GameDuration < minGameDurationSec {
			continue
		}
		fmtList = append(fmtList, gameItem)
	}
	return fmtList, nil
}

func calcUserGameScore(summonerID int64, gameSummary lcu.GameSummary) (*lcu.ScoreWithReason, error) {
	calcScoreConf := global.GetScoreConf()
	gameScore := lcu.NewScoreWithReason(defaultScore)
	var userParticipantId int
	for _, identity := range gameSummary.ParticipantIdentities {
		if identity.Player.SummonerId == summonerID {
			userParticipantId = identity.ParticipantId
		}
	}
	if userParticipantId == 0 {
		return nil, errors.New("获取用户位置失败")
	}
	var userTeamID *models.TeamID
	memberParticipantIDList := make([]int, 0, 4)
	idMapParticipant := make(map[int]lcu.Participant, len(gameSummary.Participants))
	for _, item := range gameSummary.Participants {
		if item.ParticipantId == userParticipantId {
			userTeamID = &item.TeamId
		}
		idMapParticipant[item.ParticipantId] = item
	}
	if userTeamID == nil {
		return nil, errors.New("获取用户队伍id失败")
	}
	for _, item := range gameSummary.Participants {
		if item.TeamId == *userTeamID {
			memberParticipantIDList = append(memberParticipantIDList, item.ParticipantId)
		}
	}
	totalKill := 0   // 总人头
	totalDeath := 0  // 总死亡
	totalAssist := 0 // 总助攻
	totalHurt := 0   // 总伤害
	totalMoney := 0  // 总金钱
	for _, participant := range gameSummary.Participants {
		if participant.TeamId != *userTeamID {
			continue
		}
		totalKill += participant.Stats.Kills
		totalDeath += participant.Stats.Deaths
		totalAssist += participant.Stats.Assists
		totalHurt += participant.Stats.TotalDamageDealtToChampions
		totalMoney += participant.Stats.GoldEarned
	}
	userParticipant := idMapParticipant[userParticipantId]
	isSupportRole := userParticipant.Timeline.Lane == models.LaneBottom &&
		userParticipant.Timeline.Role == models.ChampionRoleSupport

	// 五杀
	if userParticipant.Stats.PentaKills > 0 {
		gameScore.Add(calcScoreConf.PentaKills[0], lcu.ScoreOptionPentaKills)
		// 四杀
	} else if userParticipant.Stats.QuadraKills > 0 {
		gameScore.Add(calcScoreConf.QuadraKills[0], lcu.ScoreOptionQuadraKills)
		// 三杀
	} else if userParticipant.Stats.TripleKills > 0 {
		gameScore.Add(calcScoreConf.TripleKills[0], lcu.ScoreOptionTripleKills)
	}

	// 获取金钱
	if totalMoney > 0 {
		moneyRank := 1
		userMoney := userParticipant.Stats.GoldEarned
		memberMoneyList := listMemberMoney(&gameSummary, memberParticipantIDList)
		for _, v := range memberMoneyList {
			if v > userMoney {
				moneyRank++
			}
		}
		if moneyRank == 1 {
			gameScore.Add(calcScoreConf.GoldEarnedRank[0], lcu.ScoreOptionGoldEarnedRank)
		} else if moneyRank == 2 {
			gameScore.Add(calcScoreConf.GoldEarnedRank[1], lcu.ScoreOptionGoldEarnedRank)
		} else if moneyRank == 4 && !isSupportRole {
			gameScore.Add(-calcScoreConf.GoldEarnedRank[2], lcu.ScoreOptionGoldEarnedRank)
		} else if moneyRank == 5 && !isSupportRole {
			gameScore.Add(-calcScoreConf.GoldEarnedRank[3], lcu.ScoreOptionGoldEarnedRank)
		}
	}
	// 伤害占比
	if totalHurt > 0 {
		hurtRank := 1
		userHurt := userParticipant.Stats.TotalDamageDealtToChampions
		memberHurtList := listMemberHurt(&gameSummary, memberParticipantIDList)
		for _, v := range memberHurtList {
			if v > userHurt {
				hurtRank++
			}
		}
		if hurtRank == 1 {
			gameScore.Add(calcScoreConf.HurtRank[0], lcu.ScoreOptionHurtRank)
		} else if hurtRank == 2 {
			gameScore.Add(calcScoreConf.HurtRank[1], lcu.ScoreOptionHurtRank)
		}
	}
	// 金钱转换伤害比
	if totalMoney > 0 && totalHurt > 0 {
		money2hurtRateRank := 1
		userMoney2hurtRate := float64(userParticipant.Stats.TotalDamageDealtToChampions) / float64(userParticipant.Stats.
			GoldEarned)
		memberMoney2hurtRateList := listMemberMoney2hurtRate(&gameSummary, memberParticipantIDList)
		for _, v := range memberMoney2hurtRateList {
			if v > userMoney2hurtRate {
				money2hurtRateRank++
			}
		}
		if money2hurtRateRank == 1 {
			gameScore.Add(calcScoreConf.Money2hurtRateRank[0], lcu.ScoreOptionMoney2hurtRateRank)
		} else if money2hurtRateRank == 2 {
			gameScore.Add(calcScoreConf.Money2hurtRateRank[1], lcu.ScoreOptionMoney2hurtRateRank)
		}
	}
	// 视野得分
	{
		visionScoreRank := 1
		userVisionScore := userParticipant.Stats.VisionScore
		memberVisionScoreList := listMemberVisionScore(&gameSummary, memberParticipantIDList)
		for _, v := range memberVisionScoreList {
			if v > userVisionScore {
				visionScoreRank++
			}
		}
		if visionScoreRank == 1 {
			gameScore.Add(5, lcu.ScoreOptionVisionScoreRank)
		} else if visionScoreRank == 2 {
			gameScore.Add(2, lcu.ScoreOptionVisionScoreRank)
		}
	}

	// 人头占比

	if totalKill > 0 {
		userKillRate := float64(userParticipant.Stats.Kills*100) / float64(totalKill)
		if userKillRate > 25 {
			i := userKillRate - 25
			// 击杀数量达到 团队5人 (每人20%) 除开辅助 100/4=25 说明对队伍贡献大于平局 得加分
			// 每1% 计算2分
			gameScore.Add(i*2, lcu.ScoreOptionKillRate)

		} else if userKillRate < 15 && isSupportRole == false {
			i := 15 - userKillRate
			// 伤害达到 团队5人 (每人20%) 除开辅助 100/4=25 按15% 为最低分数线计算
			// 每1% 计算2分 // 不是辅助 伤害还低 扣分
			gameScore.Add(i*-2, lcu.ScoreOptionKillRate)
		}

	}
	// 伤害占比
	var userHurtRate float64 = 1
	if totalHurt > 0 {

		userHurtRate = float64(userParticipant.Stats.TotalDamageDealtToChampions*100) / float64(totalHurt)

		if userHurtRate > 25 {
			i := userHurtRate - 25
			// 伤害达到 团队5人 (每人20%) 除开辅助 100/4=25 说明对队伍贡献大于平局 得加分
			// 每1% 计算2分 // 通常伤害高的人 人头也高 其实这就相当于重复计算了
			gameScore.Add(i*2, lcu.ScoreOptionHurtRate)

		} else if userHurtRate < 15 && isSupportRole == false {
			i := 15 - userHurtRate
			// 人头计算 团队5人 (每人20%) 除开辅助 100/4=25 按15% 为最低分数线计算
			// 每1% 计算2分 // 不是辅助 人头也没有 扣分处理
			gameScore.Add(i*-2, lcu.ScoreOptionHurtRate)
		}

	}
	// 助攻占比
	if totalAssist > 0 {

		userAssistRate := float64(userParticipant.Stats.Assists*100) / float64(totalAssist)
		if userAssistRate > 25 {
			i := userAssistRate - 25
			// 助攻到 团队5人 (每人20%) 除开辅助 100/4=25 说明对队伍贡献大于平局 得加分
			// 每1% 计算1分 // 通常伤害高的人 人头也高 其实这就相当于重复计算了
			gameScore.Add(i/2, lcu.ScoreOptionAssistRate)
		}
	}
	// 死亡占比 （负面评价）
	// 允许伤害 > 10 死亡次数不是0 的玩家进行运算、
	// 如果不是辅助 且伤害<10% 扣分 已在伤害占比中扣除
	if userParticipant.Stats.Deaths > 0 && userHurtRate > 10 {
		// 获取团队死亡总数
		var teamDeaths = 0
		for _, v := range gameSummary.Participants {
			if v.TeamId == *userTeamID {
				teamDeaths = teamDeaths + v.Stats.Deaths
			}
		}
		// 死亡数据所有人平分 5人均摊
		deathsAllowed := math.Floor(float64(teamDeaths / 5))
		i := float64(userParticipant.Stats.Deaths) - deathsAllowed
		if i > 0 {
			// 死的较多 每多一次 扣10分 少一次 加5分 // 最多+20 最多扣30
			if i > 3 {
				i = 3
			}
			gameScore.Add(i*-10, lcu.ScoreOptionKDAAdjust)
		} else if i < 0 {
			// 这里i 是负数 所以得乘以-5 表示加分
			if i < -4 {
				i = -4
			}

			gameScore.Add(i*-5, lcu.ScoreOptionKDAAdjust)
		}

	} else if userParticipant.Stats.Deaths == 0 && userHurtRate > 20 {
		// 一次没死 加20 且团队贡献大于 20%
		gameScore.Add(20, lcu.ScoreOptionKDAAdjust)
	}

	kdaInfoStr := fmt.Sprintf("%d/%d/%d", userParticipant.Stats.Kills, userParticipant.Stats.Deaths,
		userParticipant.Stats.Assists)
	if global.IsDevMode() {
		log.Printf("对局%d得分:%.2f, kda:%s,原因:%s", gameSummary.GameId, gameScore.Value(), kdaInfoStr, gameScore.Reasons2String())
	}
	return gameScore, nil
}

func listMemberVisionScore(gameSummary *lcu.GameSummary, memberParticipantIDList []int) []int {
	res := make([]int, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArrayInt(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, participant.Stats.VisionScore)
	}
	return res
}

func listMemberMoney2hurtRate(gameSummary *lcu.GameSummary, memberParticipantIDList []int) []float64 {
	res := make([]float64, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArrayInt(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, float64(participant.Stats.TotalDamageDealtToChampions)/float64(participant.Stats.
			GoldEarned))
	}
	return res
}

func listMemberMoney(gameSummary *lcu.GameSummary, memberParticipantIDList []int) []int {
	res := make([]int, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArrayInt(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, participant.Stats.GoldEarned)
	}
	return res
}

func listMemberHurt(gameSummary *lcu.GameSummary, memberParticipantIDList []int) []int {
	res := make([]int, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArrayInt(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, participant.Stats.TotalDamageDealtToChampions)
	}
	return res
}
func getAllUsersFromSession(selfID int64, session *lcu.GameFlowSession) (selfTeamUsers []int64,
	enemyTeamUsers []int64) {
	selfTeamUsers = make([]int64, 0, 5)
	enemyTeamUsers = make([]int64, 0, 5)
	selfTeamID := models.TeamIDNone
	for _, teamUser := range session.GameData.TeamOne {
		summonerID := int64(teamUser.SummonerId)
		if selfID == summonerID {
			selfTeamID = models.TeamIDBlue
			break
		}
	}
	if selfTeamID == models.TeamIDNone {
		for _, teamUser := range session.GameData.TeamTwo {
			summonerID := int64(teamUser.SummonerId)
			if selfID == summonerID {
				selfTeamID = models.TeamIDRed
				break
			}
		}
	}
	if selfTeamID == models.TeamIDNone {
		return
	}
	for _, user := range session.GameData.TeamOne {
		userID := int64(user.SummonerId)
		if userID <= 0 {
			return
		}
		if models.TeamIDBlue == selfTeamID {
			selfTeamUsers = append(selfTeamUsers, userID)
		} else {
			enemyTeamUsers = append(enemyTeamUsers, userID)
		}
	}
	for _, user := range session.GameData.TeamTwo {
		userID := int64(user.SummonerId)
		if userID <= 0 {
			return
		}
		if models.TeamIDRed == selfTeamID {
			selfTeamUsers = append(selfTeamUsers, userID)
		} else {
			enemyTeamUsers = append(enemyTeamUsers, userID)
		}
	}
	return
}
