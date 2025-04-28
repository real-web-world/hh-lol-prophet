package lcu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/real-web-world/hh-lol-prophet/services/lcu/models"
	"github.com/real-web-world/hh-lol-prophet/services/logger"
)

const (
	JoinedRoomMsg                                         = "joined_room"
	ConversationMsgTypeSystem models.ConversationMsgType  = "system"
	ChampSelectPatchTypePick  models.ChampSelectPatchType = "pick"
	ChampSelectPatchTypeBan   models.ChampSelectPatchType = "ban"
	AvailabilityOffline       models.Availability         = "offline" // 离线
)

var (
	queryGameSummaryLimiter = rate.NewLimiter(rate.Every(time.Second/50), 50)
)

// 获取当前召唤师
func GetCurrSummoner() (*models.CurrSummoner, error) {
	bts, err := cli.httpGet("/lol-summoner/v1/current-summoner")
	if err != nil {
		return nil, err
	}
	data := &models.CurrSummoner{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("获取当前召唤师失败", zap.Error(err))
		return nil, err
	}
	if data.SummonerId == 0 {
		return nil, errors.New("获取当前召唤师失败")
	}
	return data, nil
}

// 获取比赛记录
func ListGamesBySummonerID(summonerID int64, begin, limit int) (*models.GameListResp, error) {
	bts, err := cli.httpGet(fmt.Sprintf("/lol-match-history/v3/matchlist/account/%d?begIndex=%d&endIndex=%d",
		summonerID, begin, begin+limit))
	if err != nil {
		return nil, err
	}
	data := &models.GameListResp{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("获取比赛记录", zap.Error(err))
		return nil, err
	}
	return data, nil
}

// 获取比赛记录
func ListGamesByPUUID(puuid string, begin, limit int) (*models.GameListResp, error) {
	bts, err := cli.httpGet(fmt.Sprintf("/lol-match-history/v1/products/lol/%s/matches?begIndex=%d&endIndex=%d",
		puuid, begin, begin+limit))
	if err != nil {
		return nil, err
	}
	data := &models.GameListResp{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("获取比赛记录", zap.Error(err))
		return nil, err
	}
	return data, nil
}

// 获取会话组消息记录
func ListConversationMsg(conversationID string) ([]models.ConversationMsg, error) {
	bts, err := cli.httpGet(fmt.Sprintf("/lol-chat/v1/conversations/%s/messages", conversationID))
	if err != nil {
		return nil, err
	}
	list := make([]models.ConversationMsg, 0, 10)
	err = json.Unmarshal(bts, &list)
	if err != nil {
		logger.Info("获取会话组消息记录失败", zap.Error(err))
		return nil, err
	}
	return list, nil
}

// 获取当前对局聊天组
func GetCurrConversationID() (string, error) {
	bts, err := cli.httpGet("/lol-chat/v1/conversations")
	if err != nil {
		return "", err
	}
	list := make([]models.Conversation, 0, 1)
	err = json.Unmarshal(bts, &list)
	if err != nil {
		logger.Info("获取当前对局聊天组失败", zap.Error(err))
		return "", err
	}
	for _, conversation := range list {
		if conversation.Type == models.GameStatusChampionSelect {
			return conversation.Id, nil
		}
	}
	return "", errors.New("当前不在英雄选择阶段")
}

// 发送消息到聊天组
func SendConversationMsg(msg string, conversationID string) error {
	data := struct {
		Body string `json:"body"`
		Type string `json:"type"`
	}{
		Body: msg,
		Type: "chat",
	}
	_, err := cli.httpPost(fmt.Sprintf("/lol-chat/v1/conversations/%s/messages", conversationID), data)
	return err
}

// 申请加好友
func ApplyFriend(summonerID int64) error {
	data := struct {
		ID string `json:"id"`
	}{
		ID: strconv.FormatInt(summonerID, 10),
	}
	_, err := cli.httpPost("/lol-chat/v1/friend-requests", data)
	return err
}

// 取消加好友
func CancelApplyFriend(summonerID int64) error {
	_, err := cli.httpDel(fmt.Sprintf("/lol-chat/v1/friend-requests/%d", summonerID))
	return err
}

// 查询用户信息
func ListSummoner(summonerIDList []int64) ([]models.Summoner, error) {
	idStrList := make([]string, 0, len(summonerIDList))
	for _, id := range summonerIDList {
		idStrList = append(idStrList, strconv.FormatInt(id, 10))
	}
	bts, err := cli.httpGet(fmt.Sprintf("/lol-summoner/v2/summoners?ids=[%s]",
		strings.Join(idStrList, ",")))
	if len(bts) > 0 && bts[0] == '[' {
		list := make([]models.Summoner, 0, len(summonerIDList))
		err = json.Unmarshal(bts, &list)
		if err != nil {
			logger.Info("查询用户信息失败", zap.Error(err))
			return nil, err
		}
		return list, err
	}
	data := &models.CommonResp{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("查询用户信息失败", zap.Error(err))
		return nil, err
	}
	return nil, errors.New(data.Message)
}

// 查询用户信息
func QuerySummoner(summonerID int64) (*models.Summoner, error) {
	list, err := ListSummoner([]int64{summonerID})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errors.New("获取召唤师信息失败 list == 0")
	}
	return &list[0], nil
}

// 查询对局详情
func QueryGameSummary(gameID int64) (*models.GameSummary, error) {
	_ = queryGameSummaryLimiter.Wait(context.Background())
	bts, err := cli.httpGet(fmt.Sprintf("/lol-match-history/v1/games/%d", gameID))
	if err != nil {
		return nil, err
	}
	data := &models.GameSummary{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		//logger.Info("查询对局详情失败", zap.Error(err))
		return nil, err
	}
	if data.CommonResp.ErrorCode != "" {
		return nil, errors.New(fmt.Sprintf("查询对局详情失败 :%s ,gameID: %d", data.CommonResp.Message, gameID))
	}
	return data, nil
}

// 查询用户信息
func QuerySummonerByName(name string) (*models.Summoner, error) {
	bts, err := cli.httpGet(fmt.Sprintf("/lol-summoner/v1/summoners?name=%s", url.QueryEscape(name)))
	if err != nil {
		return nil, err
	}
	data := &models.Summoner{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("搜索用户失败", zap.Error(err))
		return nil, err
	}
	if data.CommonResp.ErrorCode != "" {
		return nil, errors.New(fmt.Sprintf("搜索用户失败 :%s", data.CommonResp.Message))
	}
	return data, nil
}

// 接受对局
func AcceptGame() error {
	_, err := cli.httpPost("/lol-matchmaking/v1/ready-check/accept", nil)
	return err
}

// 获取选人会话
func GetChampSelectSession() (*models.ChampSelectSessionInfo, error) {
	bts, err := cli.httpGet("/lol-champ-select/v1/session")
	if err != nil {
		return nil, err
	}
	data := &models.ChampSelectSessionInfo{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("查询选人会话详情失败", zap.Error(err))
		return nil, err
	}
	if data.CommonResp.ErrorCode != "" {
		return nil, errors.New(fmt.Sprintf("查询选人会话详情失败 :%s", data.CommonResp.Message))
	}
	return data, nil
}

func ChampSelectPatchAction(championID, actionID int, patchType *models.ChampSelectPatchType,
	completed *bool) error {
	body := struct {
		Completed  *bool                        `json:"completed,omitempty"`
		Type       *models.ChampSelectPatchType `json:"type,omitempty"`
		ChampionID int                          `json:"championId"`
	}{
		Completed:  completed,
		Type:       patchType,
		ChampionID: championID,
	}
	bts, err := cli.httpPatch(fmt.Sprintf("/lol-champ-select/v1/session/actions/%d", actionID), body)
	if err != nil {
		return err
	}
	if len(bts) == 0 {
		return nil
	}
	data := &models.CommonResp{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("ChampSelectPatchAction详情失败", zap.Error(err), zap.Any("completed", completed),
			zap.Any("patchType", patchType), zap.Int("championID", championID), zap.ByteString("bts", bts))
		return err
	}
	if data.ErrorCode != "" {
		return errors.New(fmt.Sprintf("ChampSelectPatchAction失败 :%s", data.Message))
	}
	return nil
}

// 预选英雄
func PrePickChampion(championID, actionID int) error {
	return ChampSelectPatchAction(championID, actionID, nil, nil)
}

// 选择英雄
func PickChampion(championID, actionID int) error {
	patchType := new(models.ChampSelectPatchType)
	*patchType = ChampSelectPatchTypePick
	completed := new(bool)
	*completed = true
	return ChampSelectPatchAction(championID, actionID, patchType, completed)
}

// ban英雄
func BanChampion(championID, actionID int) error {
	patchType := new(models.ChampSelectPatchType)
	*patchType = ChampSelectPatchTypeBan
	completed := new(bool)
	*completed = true
	return ChampSelectPatchAction(championID, actionID, patchType, completed)
}

// 查询游戏会话
func QueryGameFlowSession() (*models.GameFlowSession, error) {
	bts, err := cli.httpGet("/lol-gameflow/v1/session")
	if err != nil {
		return nil, err
	}
	data := &models.GameFlowSession{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("查询游戏会话失败", zap.Error(err))
		return nil, err
	}
	if data.CommonResp.ErrorCode != "" {
		return nil, errors.New(fmt.Sprintf("查询游戏会话失败 :%s", data.CommonResp.Message))
	}
	return data, nil
}

// 更新用户信息
func UpdateSummonerProfile(updateData models.UpdateSummonerProfileData) (*models.SummonerProfileData, error) {
	bts, err := cli.req(http.MethodPut, "/lol-chat/v1/me", updateData)
	if err != nil {
		return nil, err
	}
	data := &models.SummonerProfileData{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("更新用户信息失败", zap.Error(err))
		return nil, err
	}
	if data.CommonResp.ErrorCode != "" {
		return nil, errors.New(fmt.Sprintf("更新用户信息请求失败 :%s", data.CommonResp.Message))
	}
	return data, err
}

// 设置离线状态
func SetupFakerOffline() error {
	data := models.UpdateSummonerProfileData{
		Availability: AvailabilityOffline,
	}
	_, err := UpdateSummonerProfile(data)
	return err
}

// 获取玩家简介信息
func GetSummonerProfile() (*models.SummonerProfileData, error) {
	data := models.UpdateSummonerProfileData{}
	return UpdateSummonerProfile(data)
}
