package lcu

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/real-web-world/hh-lol-prophet/global"
	"github.com/real-web-world/hh-lol-prophet/services/lcu/models"

	"github.com/real-web-world/hh-lol-prophet/pkg/bdk"
	"github.com/real-web-world/hh-lol-prophet/services/logger"
)

type (
	UserScore struct {
		SummonerID   int64    `json:"summonerID"`
		SummonerName string   `json:"summonerName"`
		Score        float64  `json:"score"`
		CurrKDA      [][3]int `json:"currKDA"`
	}
)

const (
	defaultScore = 100 // 默认分数
)

func ChampionSelectStart() {
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
	// if !false {
	// summonerIDList = []int64{2965189289, 4014052617, 4015941802, 2613569584655104, 2950744173}
	// summonerIDList = []int64{4006944917}
	// }
	logger.Debug("队伍人员列表:", zap.Any("summonerIDList", summonerIDList))
	// 查询所有用户的信息并计算得分
	g := errgroup.Group{}
	summonerIDMapScore := map[int64]UserScore{}
	mu := sync.Mutex{}
	for _, summonerID := range summonerIDList {
		summonerID := summonerID
		g.Go(func() error {
			actScore, err := GetUserScore(summonerID)
			if err != nil {
				logger.Error("计算用户得分失败", zap.Error(err), zap.Int64("summonerID", summonerID))
				return nil
			}
			mu.Lock()
			summonerIDMapScore[summonerID] = *actScore
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	// 根据所有用户的分数判断小代上等马中等马下等马
	for _, score := range summonerIDMapScore {
		log.Printf("用户:%s,得分:%.2f\n", score.SummonerName, score.Score)
	}
	scoreCfg := global.GetScoreConf()
	// 发送到选人界面
	for _, scoreInfo := range summonerIDMapScore {
		var horse string
		for _, v := range scoreCfg.Horse {
			if scoreInfo.Score >= v.Score {
				horse = v.Name
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
		msg := fmt.Sprintf("%s(%d): %s %s  -- lol.buffge点康姆", horse, int(scoreInfo.Score), scoreInfo.SummonerName,
			currKDAMsg)
		_ = SendConversationMsg(msg, conversationID)
	}
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
func getSummonerIDListFromConversationMsgList(msgList []ConversationMsg) []int64 {
	summonerIDList := make([]int64, 0, 5)
	for _, msg := range msgList {
		if msg.Type == ConversationMsgTypeSystem && msg.Body == JoinedRoomMsg {
			summonerIDList = append(summonerIDList, msg.FromSummonerId)
		}
	}
	return summonerIDList
}

func GetUserScore(summonerID int64) (*UserScore, error) {
	userScoreInfo := &UserScore{
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
	gameSummaryList := make([]GameSummary, 0, len(gameList))
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
			gameSummary, err := QueryGameSummary(info.GameId)
			if err != nil {
				logger.Error("获取游戏对局详细信息失败", zap.Error(err), zap.Int64("id", info.GameId))
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
			logger.Error("游戏战绩计算用户得分失败", zap.Error(err), zap.Int64("summonerID", summonerID),
				zap.Int64("gameID", gameSummary.GameId))
			return userScoreInfo, nil
		}
		weightScoreItem := gameScoreWithWeight{
			score:       gameScore,
			isCurrTimes: nowTime.Before(gameSummary.GameCreationDate.Add(time.Hour * 5)),
		}
		if weightScoreItem.isCurrTimes {
			currTimeScoreList = append(currTimeScoreList, gameScore)
		} else {

			otherGameScoreList = append(otherGameScoreList, gameScore)
		}
		totalGameCount++
		totalScore += gameScore
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

func listGameHistory(summonerID int64) ([]GameInfo, error) {
	fmtList := make([]GameInfo, 0, 20)
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
		fmtList = append(fmtList, gameItem)
	}
	return fmtList, nil
}

func calcUserGameScore(summonerID int64, gameSummary GameSummary) (float64, error) {
	calcScoreConf := global.GetScoreConf()
	gameScore := float64(defaultScore)
	var userParticipantId int
	for _, identity := range gameSummary.ParticipantIdentities {
		if identity.Player.SummonerId == summonerID {
			userParticipantId = identity.ParticipantId
		}
	}
	if userParticipantId == 0 {
		return 0, errors.New("获取用户位置失败")
	}
	var userTeamID *models.TeamID
	memberParticipantIDList := make([]int, 0, 4)
	idMapParticipant := make(map[int]Participant, len(gameSummary.Participants))
	for _, item := range gameSummary.Participants {
		if item.ParticipantId == userParticipantId {
			userTeamID = &item.TeamId
		}
		idMapParticipant[item.ParticipantId] = item
	}
	if userTeamID == nil {
		return 0, errors.New("获取用户队伍id失败")
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
		gameScore += calcScoreConf.FirstBlood[0]
		// 一血助攻
	} else if userParticipant.Stats.FirstBloodAssist {
		gameScore += calcScoreConf.FirstBlood[0]
	}
	// 五杀
	if userParticipant.Stats.PentaKills > 0 {
		gameScore += calcScoreConf.PentaKills[0]
		// 四杀
	} else if userParticipant.Stats.QuadraKills > 0 {
		gameScore += calcScoreConf.QuadraKills[0]
		// 三杀
	} else if userParticipant.Stats.TripleKills > 0 {
		gameScore += calcScoreConf.TripleKills[0]
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
			gameScore += calcScoreConf.JoinTeamRateRank[0]
		} else if joinTeamRateRank == 2 {
			gameScore += calcScoreConf.JoinTeamRateRank[1]
		} else if joinTeamRateRank == 4 {
			gameScore -= calcScoreConf.JoinTeamRateRank[2]
		} else if joinTeamRateRank == 5 {
			gameScore -= calcScoreConf.JoinTeamRateRank[3]
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
			gameScore += calcScoreConf.GoldEarnedRank[0]
		} else if moneyRank == 2 {
			gameScore += calcScoreConf.GoldEarnedRank[1]
		} else if moneyRank == 4 && !isSupportRole {
			gameScore -= calcScoreConf.GoldEarnedRank[2]
		} else if moneyRank == 5 && !isSupportRole {
			gameScore -= calcScoreConf.GoldEarnedRank[3]
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
			gameScore += calcScoreConf.HurtRank[0]
		} else if hurtRank == 2 {
			gameScore += calcScoreConf.HurtRank[1]
		}
	}
	// 金钱转换伤害比 todo 是否跟伤害占比重复 感觉可以改下
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
			gameScore += calcScoreConf.Money2hurtRateRank[0]
		} else if money2hurtRateRank == 2 {
			gameScore += calcScoreConf.Money2hurtRateRank[1]
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
			gameScore += calcScoreConf.VisionScoreRank[0]
		} else if visionScoreRank == 2 {
			gameScore += calcScoreConf.VisionScoreRank[1]
		}
	}
	// 补兵 每分钟8个刀以上加5分 ,9+10, 10+20
	{
		totalMinionsKilled := userParticipant.Stats.TotalMinionsKilled
		gameDurationMinute := gameSummary.GameDuration / 60
		minuteMinionsKilled := totalMinionsKilled / gameDurationMinute
		for _, minionsKilledLimit := range calcScoreConf.MinionsKilled {
			if minuteMinionsKilled >= int(minionsKilledLimit[0]) {
				gameScore += minionsKilledLimit[1]
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
			if userKillRate > killRateConfItem.KillRateLimit {
			killRateConfItemLoop:
				for _, limitConf := range killRateConfItem.ScoreConf {
					if userParticipant.Stats.Kills > int(limitConf[0]) {
						gameScore += limitConf[1]
						break killRateConfItemLoop
					}
				}
				break userKillRateLoop
			}
		}
	}
	// 伤害占比
	if totalHurt > 0 {
		// 伤害占比>50%
		userHurtRate := float64(userParticipant.Stats.TotalDamageDealtToChampions) / float64(totalHurt)
	userHurtRateLoop:
		for _, killRateConfItem := range calcScoreConf.HurtRate {
			if userHurtRate > killRateConfItem.HurtRateLimit {
			hurtRateConfItemLoop:
				for _, limitConf := range killRateConfItem.ScoreConf {
					if userParticipant.Stats.Kills > int(limitConf[0]) {
						gameScore += limitConf[1]
						break hurtRateConfItemLoop
					}
				}
				break userHurtRateLoop
			}
		}
	}
	// 助攻占比
	if totalAssist > 0 {
		// 助攻占比>50%
		userAssistRate := float64(userParticipant.Stats.Assists) / float64(totalAssist)
	userAssistRateLoop:
		for _, killRateConfItem := range calcScoreConf.AssistRate {
			if userAssistRate > killRateConfItem.AssistRateLimit {
			assistRateConfItemLoop:
				for _, limitConf := range killRateConfItem.ScoreConf {
					if userParticipant.Stats.Kills > int(limitConf[0]) {
						gameScore += limitConf[1]
						break assistRateConfItemLoop
					}
				}
				break userAssistRateLoop
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
	gameScore += (float64(userParticipant.Stats.Kills+userParticipant.Stats.Assists)/float64(userDeathTimes) -
		calcScoreConf.AdjustKDA[0] +
		float64(userParticipant.Stats.Kills-userParticipant.Stats.Deaths)/calcScoreConf.AdjustKDA[1]) * userJoinTeamKillRate
	// log.Printf("game: %d,kda: %d/%d/%d\n", gameSummary.GameId, userParticipant.Stats.Kills,
	// 	userParticipant.Stats.Deaths, userParticipant.Stats.Assists)
	return gameScore, nil
}

func listMemberVisionScore(gameSummary *GameSummary, memberParticipantIDList []int) []int {
	res := make([]int, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArrayInt(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, participant.Stats.VisionScore)
	}
	return res
}

func listMemberMoney2hurtRate(gameSummary *GameSummary, memberParticipantIDList []int) []float64 {
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

func listMemberMoney(gameSummary *GameSummary, memberParticipantIDList []int) []int {
	res := make([]int, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArrayInt(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, participant.Stats.GoldEarned)
	}
	return res
}

func listMemberJoinTeamKillRates(gameSummary *GameSummary, totalKill int, memberParticipantIDList []int) []float64 {
	res := make([]float64, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArrayInt(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, float64(participant.Stats.Assists+participant.Stats.Kills)/float64(
			totalKill))
	}
	return res
}

func listMemberHurt(gameSummary *GameSummary, memberParticipantIDList []int) []int {
	res := make([]int, 0, 4)
	for _, participant := range gameSummary.Participants {
		if !bdk.InArrayInt(participant.ParticipantId, memberParticipantIDList) {
			continue
		}
		res = append(res, participant.Stats.TotalDamageDealtToChampions)
	}
	return res
}
