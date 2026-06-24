package lcu

import "encoding/json"

type (
	WsEvt string
	WsMsg struct {
		Data      json.RawMessage `json:"data"`
		EventType string          `json:"event_type"`
		Uri       string          `json:"uri"`
	}
)

// ws msg
var (
	// 订阅所有事件
	SubscribeAllEventMsg = []byte("[5, \"OnJsonApiEvent\"]")
)

// lcu ws
const (
	OnJsonApiEventPrefixLen = len(`[8,"OnJsonApiEvent",`)
)

// WsEvt
const (
	WsEvtGameFlowChanged          WsEvt = "/lol-gameflow/v1/gameflow-phase" // 游戏状态切换
	WsEvtChampSelectUpdateSession WsEvt = "/lol-champ-select/v1/session"    // 进入英雄选择阶段
)
