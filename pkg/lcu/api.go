package lcu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/real-web-world/hh-lol-prophet/pkg/lcu/model"
)

const (
	JoinedRoomMsg                                        = "joined_room"
	ConversationMsgTypeSystem model.ConversationMsgType  = "system"
	ChampSelectPatchTypePick  model.ChampSelectPatchType = "pick"
	ChampSelectPatchTypeBan   model.ChampSelectPatchType = "ban"
	AvailabilityOffline       model.Availability         = "offline" // 离线
)

var queryGameSummaryLimiter = rate.NewLimiter(rate.Every(time.Second/50), 50)

// 获取当前召唤师
func (cli Client) GetCurrSummoner() (*model.CurrSummoner, error) {
	bts, err := cli.httpGet("/lol-summoner/v1/current-summoner")
	if err != nil {
		return nil, err
	}
	data := &model.CurrSummoner{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		cli.l.Debug("获取当前召唤师失败", slog.Any("err", err))
		return nil, err
	}
	if data.SummonerId == 0 {
		return nil, errors.New("获取当前召唤师失败")
	}
	return data, nil
}

// 获取比赛记录
func (cli Client) ListGamesBySummonerID(summonerID int64, begin, limit int) (*model.GameListResp, error) {
	bts, err := cli.httpGet(fmt.Sprintf("/lol-match-history/v3/matchlist/account/%d?begIndex=%d&endIndex=%d",
		summonerID, begin, begin+limit))
	if err != nil {
		return nil, err
	}
	data := &model.GameListResp{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		cli.l.Debug("获取比赛记录", slog.Any("err", err))
		return nil, err
	}
	return data, nil
}

// 获取比赛记录
func (cli Client) ListGamesByPUUID(puuid string, begin, limit int) (*model.GameListResp, error) {
	bts, err := cli.httpGet(fmt.Sprintf("/lol-match-history/v1/products/lol/%s/matches?begIndex=%d&endIndex=%d",
		puuid, begin, begin+limit))
	if err != nil {
		return nil, err
	}
	data := &model.GameListResp{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		cli.l.Debug("获取比赛记录", slog.Any("err", err))
		return nil, err
	}
	return data, nil
}

// 获取会话组消息记录
func (cli Client) ListConversationMsg(conversationID string) ([]model.ConversationMsg, error) {
	bts, err := cli.httpGet(fmt.Sprintf("/lol-chat/v1/conversations/%s/messages", conversationID))
	if err != nil {
		return nil, err
	}
	list := make([]model.ConversationMsg, 0, 10)
	err = json.Unmarshal(bts, &list)
	if err != nil {
		cli.l.Debug("获取会话组消息记录失败", slog.Any("err", err))
		return nil, err
	}
	return list, nil
}

// 获取当前对局聊天组
func (cli Client) GetCurrConversationID() (string, error) {
	bts, err := cli.httpGet("/lol-chat/v1/conversations")
	if err != nil {
		return "", err
	}
	list := make([]model.Conversation, 0, 1)
	err = json.Unmarshal(bts, &list)
	if err != nil {
		cli.l.Debug("获取当前对局聊天组失败", slog.Any("err", err))
		return "", err
	}
	for _, conversation := range list {
		if conversation.Type == model.GameStatusChampionSelect {
			return conversation.Id, nil
		}
	}
	return "", errors.New("当前不在英雄选择阶段")
}

// 发送消息到聊天组
func (cli Client) SendConversationMsg(msg string, conversationID string) error {
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
func (cli Client) ApplyFriend(summonerID int64) error {
	data := struct {
		ID string `json:"id"`
	}{
		ID: strconv.FormatInt(summonerID, 10),
	}
	_, err := cli.httpPost("/lol-chat/v1/friend-requests", data)
	return err
}

// 取消加好友
func (cli Client) CancelApplyFriend(summonerID int64) error {
	_, err := cli.httpDel(fmt.Sprintf("/lol-chat/v1/friend-requests/%d", summonerID))
	return err
}

// 查询用户信息
func (cli Client) ListSummoner(summonerIDList []int64) ([]model.Summoner, error) {
	idStrList := make([]string, 0, len(summonerIDList))
	for _, id := range summonerIDList {
		idStrList = append(idStrList, strconv.FormatInt(id, 10))
	}
	bts, err := cli.httpGet(fmt.Sprintf("/lol-summoner/v2/summoners?ids=[%s]",
		strings.Join(idStrList, ",")))
	if err != nil {
		return nil, err
	}
	if len(bts) > 0 && bts[0] == '[' {
		list := make([]model.Summoner, 0, len(summonerIDList))
		err = json.Unmarshal(bts, &list)
		if err != nil {
			cli.l.Debug("查询用户信息失败", slog.Any("err", err))
			return nil, err
		}
		return list, err
	}
	data := &model.CommonResp{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		cli.l.Debug("查询用户信息失败", slog.Any("err", err))
		return nil, err
	}
	return nil, errors.New(data.Message)
}

// 查询用户信息
func (cli Client) QuerySummoner(summonerID int64) (*model.Summoner, error) {
	list, err := cli.ListSummoner([]int64{summonerID})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errors.New("获取召唤师信息失败 list == 0")
	}
	return &list[0], nil
}

// 查询对局详情
func (cli Client) QueryGameSummary(gameID int64) (*model.GameSummary, error) {
	_ = queryGameSummaryLimiter.Wait(context.Background())
	bts, err := cli.httpGet(fmt.Sprintf("/lol-match-history/v1/games/%d", gameID))
	if err != nil {
		return nil, err
	}
	data := &model.GameSummary{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		// cli.l.Debug("查询对局详情失败", slog.Any("err", err))
		return nil, err
	}
	if data.ErrorCode != "" {
		return nil, fmt.Errorf("查询对局详情失败 :%s ,gameID: %d", data.Message, gameID)
	}
	return data, nil
}

// 查询用户信息
func (cli Client) QuerySummonerByName(name string) (*model.Summoner, error) {
	bts, err := cli.httpGet(fmt.Sprintf("/lol-summoner/v1/summoners?name=%s", url.QueryEscape(name)))
	if err != nil {
		return nil, err
	}
	data := &model.Summoner{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		cli.l.Debug("搜索用户失败", slog.Any("err", err))
		return nil, err
	}
	if data.ErrorCode != "" {
		return nil, fmt.Errorf("搜索用户失败 :%s", data.Message)
	}
	return data, nil
}

// 接受对局
func (cli Client) AcceptGame() error {
	_, err := cli.httpPost("/lol-matchmaking/v1/ready-check/accept", nil)
	return err
}

// 获取选人会话
func (cli Client) GetChampSelectSession() (*model.ChampSelectSessionInfo, error) {
	bts, err := cli.httpGet("/lol-champ-select/v1/session")
	if err != nil {
		return nil, err
	}
	data := &model.ChampSelectSessionInfo{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		cli.l.Debug("查询选人会话详情失败", slog.Any("err", err))
		return nil, err
	}
	if data.ErrorCode != "" {
		return nil, fmt.Errorf("查询选人会话详情失败 :%s", data.Message)
	}
	return data, nil
}

func (cli Client) ChampSelectPatchAction(championID, actionID int, patchType *model.ChampSelectPatchType,
	completed *bool,
) error {
	body := struct {
		Completed  *bool                       `json:"completed,omitempty"`
		Type       *model.ChampSelectPatchType `json:"type,omitempty"`
		ChampionID int                         `json:"championId"`
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
	data := &model.CommonResp{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		cli.l.Debug("ChampSelectPatchAction详情失败", slog.Any("err", err), slog.Any("completed", completed),
			slog.Any("patchType", patchType), slog.Int("championID", championID),
			slog.String("bts", string(bts)))
		return err
	}
	if data.ErrorCode != "" {
		return fmt.Errorf("ChampSelectPatchAction失败 :%s", data.Message)
	}
	return nil
}

// 预选英雄
func (cli Client) PrePickChampion(championID, actionID int) error {
	return cli.ChampSelectPatchAction(championID, actionID, nil, nil)
}

// 选择英雄
func (cli Client) PickChampion(championID, actionID int) error {
	patchType := new(model.ChampSelectPatchType)
	*patchType = ChampSelectPatchTypePick
	completed := new(bool)
	*completed = true
	return cli.ChampSelectPatchAction(championID, actionID, patchType, completed)
}

// ban英雄
func (cli Client) BanChampion(championID, actionID int) error {
	patchType := new(model.ChampSelectPatchType)
	*patchType = ChampSelectPatchTypeBan
	completed := new(bool)
	*completed = true
	return cli.ChampSelectPatchAction(championID, actionID, patchType, completed)
}

// 查询游戏会话
func (cli Client) QueryGameFlowSession() (*model.GameFlowSession, error) {
	bts, err := cli.httpGet("/lol-gameflow/v1/session")
	if err != nil {
		return nil, err
	}
	data := &model.GameFlowSession{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		cli.l.Debug("查询游戏会话失败", slog.Any("err", err))
		return nil, err
	}
	if data.ErrorCode != "" {
		return nil, fmt.Errorf("查询游戏会话失败 :%s", data.Message)
	}
	return data, nil
}

// 更新用户信息
func (cli Client) UpdateSummonerProfile(
	updateData model.UpdateSummonerProfileData,
) (*model.SummonerProfileData, error) {
	bts, err := cli.req(http.MethodPut, "/lol-chat/v1/me", updateData)
	if err != nil {
		return nil, err
	}
	data := &model.SummonerProfileData{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		cli.l.Debug("更新用户信息失败", slog.Any("err", err))
		return nil, err
	}
	if data.ErrorCode != "" {
		return nil, fmt.Errorf("更新用户信息请求失败 :%s", data.Message)
	}
	return data, err
}

// 设置离线状态
func (cli Client) SetupFakerOffline() error {
	data := model.UpdateSummonerProfileData{
		Availability: AvailabilityOffline,
	}
	_, err := cli.UpdateSummonerProfile(data)
	return err
}

// 获取玩家简介信息
func (cli Client) GetSummonerProfile() (*model.SummonerProfileData, error) {
	data := model.UpdateSummonerProfileData{}
	return cli.UpdateSummonerProfile(data)
}
