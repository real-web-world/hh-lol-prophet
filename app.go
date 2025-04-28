package hh_lol_prophet

import (
	"fmt"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/pkg/errors"
	"github.com/real-web-world/bdk"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/real-web-world/hh-lol-prophet/global"
	"github.com/real-web-world/hh-lol-prophet/services/lcu"
	"github.com/real-web-world/hh-lol-prophet/services/lcu/models"
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
)

func listSummoner(summonerIDList []int64) (map[int64]*models.Summoner, error) {
	list, err := lcu.ListSummoner(summonerIDList)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]*models.Summoner, len(summonerIDList))
	for _, summoner := range list {
		summoner := summoner
		res[summoner.SummonerId] = &summoner
	}
	return res, nil
}
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
func getSummonerIDListFromConversationMsgList(msgList []models.ConversationMsg) []int64 {
	summonerIDList := make([]int64, 0, 5)
	for _, msg := range msgList {
		if msg.Type == lcu.ConversationMsgTypeSystem && msg.Body == lcu.JoinedRoomMsg && msg.FromSummonerId > 0 {
			summonerIDList = append(summonerIDList, msg.FromSummonerId)
		}
	}
	return summonerIDList
}
func GetUserScore(summoner *models.Summoner) (*lcu.UserScore, error) {
	summonerID := summoner.SummonerId
	userScoreInfo := &lcu.UserScore{
		SummonerID: summonerID,
		Score:      defaultScore,
	}
	userScoreInfo.SummonerName = fmt.Sprintf("%s#%s", summoner.GameName, summoner.TagLine)
	// 获取战绩列表
	gameList, err := listGameHistory(summoner.Puuid)
	if err != nil {
		logger.Error("获取用户战绩失败", zap.Error(err), zap.Int64("id", summonerID))
		return userScoreInfo, nil
	}
	// 获取每一局战绩
	g := errgroup.Group{}
	gameSummaryList := make([]models.GameSummary, 0, len(gameList))
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
			var gameSummary *models.GameSummary
			err = retry.Do(func() error {
				var tmpErr error
				gameSummary, tmpErr = QueryGameSummary(info.GameId)
				return tmpErr
			}, retry.Delay(time.Millisecond*10), retry.Attempts(5))
			if err != nil {
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

func listGameHistory(puuid string) ([]models.GameInfo, error) {
	scoreCfg := global.GetScoreConf()
	limit := 20
	fmtList := make([]models.GameInfo, 0, limit)
	resp, err := lcu.ListGamesByPUUID(puuid, 0, limit)
	if err != nil {
		logger.Error("查询用户战绩失败", zap.Error(err), zap.String("puuid", puuid))
		return nil, err
	}
	allowQueueIDSet := make(map[int]struct{}, len(scoreCfg.AllowQueueIDList))
	for _, v := range scoreCfg.AllowQueueIDList {
		allowQueueIDSet[v] = struct{}{}
	}
	for _, gameItem := range resp.Games.Games {
		queueID := int(gameItem.QueueId)
		if _, exist := allowQueueIDSet[queueID]; !exist {
			continue
		}
		if gameItem.GameDuration < scoreCfg.GameMinDuration {
			continue
		}
		fmtList = append(fmtList, gameItem)
	}
	gameCount := len(fmtList)
	for i := 0; i < gameCount/2; i++ {
		fmtList[i], fmtList[gameCount-1-i] = fmtList[gameCount-1-i], fmtList[i]
	}
	return fmtList, nil
}

func calcUserGameScore(summonerID int64, gameSummary models.GameSummary) (*lcu.ScoreWithReason, error) {
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
	idMapParticipant := make(map[int]models.Participant, len(gameSummary.Participants))
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
	// 一血击杀
	if userParticipant.Stats.FirstBloodKill {
		gameScore.Add(calcScoreConf.FirstBlood[0], lcu.ScoreOptionFirstBloodKill)
		// 一血助攻
	} else if userParticipant.Stats.FirstBloodAssist {
		gameScore.Add(calcScoreConf.FirstBlood[1], lcu.ScoreOptionFirstBloodAssist)
	}
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
	// 参团率
	if totalKill > 0 {
		joinTeamRateRank := 1
		userJoinTeamKillRate := float64(userParticipant.Stats.Assists+userParticipant.Stats.Kills) / float64(
			totalKill)
		memberJoinTeamKillRates := listMemberJoinTeamKillRates(&gameSummary, totalKill, memberParticipantIDList)
		for _, rate := range memberJoinTeamKillRates {
			if rate > userJoinTeamKillRate {
				joinTeamRateRank++
			}
		}
		if joinTeamRateRank == 1 {
			gameScore.Add(calcScoreConf.JoinTeamRateRank[0], lcu.ScoreOptionJoinTeamRateRank)
		} else if joinTeamRateRank == 2 {
			gameScore.Add(calcScoreConf.JoinTeamRateRank[1], lcu.ScoreOptionJoinTeamRateRank)
		} else if joinTeamRateRank == 4 {
			gameScore.Add(-calcScoreConf.JoinTeamRateRank[2], lcu.ScoreOptionJoinTeamRateRank)
		} else if joinTeamRateRank == 5 {
			gameScore.Add(-calcScoreConf.JoinTeamRateRank[3], lcu.ScoreOptionJoinTeamRateRank)
		}
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
			gameScore.Add(calcScoreConf.VisionScoreRank[0], lcu.ScoreOptionVisionScoreRank)
		} else if visionScoreRank == 2 {
			gameScore.Add(calcScoreConf.VisionScoreRank[1], lcu.ScoreOptionVisionScoreRank)
		}
	}
	// 补兵 每分钟8个刀以上加5分 ,9+10, 10+20
	{
		totalMinionsKilled := userParticipant.Stats.TotalMinionsKilled
		gameDurationMinute := gameSummary.GameDuration / 60
		minuteMinionsKilled := totalMinionsKilled / gameDurationMinute
		for _, minionsKilledLimit := range calcScoreConf.MinionsKilled {
			if minuteMinionsKilled >= int(minionsKilledLimit[0]) {
				gameScore.Add(minionsKilledLimit[1], lcu.ScoreOptionMinionsKilled)
				break
			}
		}
	}
	// 人头占比
	if totalKill > 0 {
		// 人头占比>50%
		userKillRate := float64(userParticipant.Stats.Kills) / float64(totalKill)
	userKillRateLoop:
		for _, killRateConfItem := range calcScoreConf.KillRate {
			if userKillRate > killRateConfItem.Limit {
				for _, limitConf := range killRateConfItem.ScoreConf {
					if userParticipant.Stats.Kills > int(limitConf[0]) {
						gameScore.Add(limitConf[1], lcu.ScoreOptionKillRate)
						break userKillRateLoop
					}
				}
			}
		}
	}
	// 伤害占比
	if totalHurt > 0 {
		// 伤害占比>50%
		userHurtRate := float64(userParticipant.Stats.TotalDamageDealtToChampions) / float64(totalHurt)
	userHurtRateLoop:
		for _, killRateConfItem := range calcScoreConf.HurtRate {
			if userHurtRate > killRateConfItem.Limit {
				for _, limitConf := range killRateConfItem.ScoreConf {
					if userParticipant.Stats.Kills > int(limitConf[0]) {
						gameScore.Add(limitConf[1], lcu.ScoreOptionHurtRate)
						break userHurtRateLoop
					}
				}
			}
		}
	}
	// 助攻占比
	if totalAssist > 0 {
		// 助攻占比>50%
		userAssistRate := float64(userParticipant.Stats.Assists) / float64(totalAssist)
	userAssistRateLoop:
		for _, killRateConfItem := range calcScoreConf.AssistRate {
			if userAssistRate > killRateConfItem.Limit {
				for _, limitConf := range killRateConfItem.ScoreConf {
					if userParticipant.Stats.Kills > int(limitConf[0]) {
						gameScore.Add(limitConf[1], lcu.ScoreOptionAssistRate)
						break userAssistRateLoop
					}
				}
			}
		}
	}
	userJoinTeamKillRate := 1.0
	if totalKill > 0 {
		userJoinTeamKillRate = float64(userParticipant.Stats.Assists+userParticipant.Stats.Kills) / float64(
			totalKill)
	}
	userDeathTimes := userParticipant.Stats.Deaths
	if userParticipant.Stats.Deaths == 0 {
		userDeathTimes = 1
	}
	adjustVal := (float64(userParticipant.Stats.Kills+userParticipant.Stats.Assists)/float64(userDeathTimes) -
		calcScoreConf.AdjustKDA[0] +
		float64(userParticipant.Stats.Kills-userParticipant.Stats.Deaths)/calcScoreConf.AdjustKDA[1]) * userJoinTeamKillRate
	// log.Printf("game: %d,kda: %d/%d/%d\n", gameSummary.GameId, userParticipant.Stats.Kills,
	// 	userParticipant.Stats.Deaths, userParticipant.Stats.Assists)
	gameScore.Add(adjustVal, lcu.ScoreOptionKDAAdjust)
	// kdaInfoStr := fmt.Sprintf("%d/%d/%d", userParticipant.Stats.Kills, userParticipant.Stats.Deaths,
	// 	userParticipant.Stats.Assists)
	// if global.IsDevMode() {
	// 	log.Printf("对局%d得分:%.2f, kda:%s,原因:%s", gameSummary.GameId, gameScore.Value(), kdaInfoStr, gameScore.Reasons2String())
	// }
	return gameScore, nil
}

func listMemberVisionScore(gameSummary *models.GameSummary, memberParticipantIDList []int) []int {
	res := make([]int, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArray(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, participant.Stats.VisionScore)
	}
	return res
}

func listMemberMoney2hurtRate(gameSummary *models.GameSummary, memberParticipantIDList []int) []float64 {
	res := make([]float64, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArray(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, float64(participant.Stats.TotalDamageDealtToChampions)/float64(participant.Stats.
			GoldEarned))
	}
	return res
}

func listMemberMoney(gameSummary *models.GameSummary, memberParticipantIDList []int) []int {
	res := make([]int, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArray(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, participant.Stats.GoldEarned)
	}
	return res
}

func listMemberJoinTeamKillRates(gameSummary *models.GameSummary, totalKill int, memberParticipantIDList []int) []float64 {
	res := make([]float64, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArray(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, float64(participant.Stats.Assists+participant.Stats.Kills)/float64(
			totalKill))
	}
	return res
}

func listMemberHurt(gameSummary *models.GameSummary, memberParticipantIDList []int) []int {
	res := make([]int, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArray(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, participant.Stats.TotalDamageDealtToChampions)
	}
	return res
}
func getAllUsersFromSession(selfID int64, session *models.GameFlowSession) (selfTeamUsers []int64,
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
