package hh_lol_prophet

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/real-web-world/hh-lol-prophet/global"

	"github.com/real-web-world/hh-lol-prophet/services/lcu"
	"github.com/real-web-world/hh-lol-prophet/services/lcu/models"
	"github.com/real-web-world/hh-lol-prophet/services/logger"
)

type (
	Prophet struct {
		ctx       context.Context
		lcuPort   int
		lcuToken  string
		lcuActive bool
		cancel    func()
		mu        *sync.Mutex
	}
	wsMsg struct {
		Data      interface{} `json:"data"`
		EventType string      `json:"event_type"`
		Uri       string      `json:"uri"`
	}
)

const (
	onJsonApiEventPrefixLen = len(`[8,"OnJsonApiEvent",`)
	gameFlowChangedEvt      = "/lol-gameflow/v1/gameflow-phase"
)

func NewProphet() *Prophet {
	ctx, cancel := context.WithCancel(context.Background())
	return &Prophet{
		ctx:    ctx,
		cancel: cancel,
		mu:     &sync.Mutex{},
	}
}
func (p Prophet) Run() error {
	go p.MonitorStart()
	go func() {
		for i := 0; i < 5; i++ {
			if global.GetUserInfo().CpuID != "" {
				break
			}
			time.Sleep(time.Second * 2)
		}
		sentry.CaptureMessage("lol对局先知已启动")
	}()
	log.Printf("lol对局先知已启动 v%s -- lol.buffge.com", APPVersion)
	return p.notifyQuit()
}
func (p Prophet) isLcuActive() bool {
	return p.lcuActive
}
func (p Prophet) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	// stop all task
	return nil
}
func (p Prophet) MonitorStart() {
	for {
		if !p.isLcuActive() {
			port, token, err := lcu.GetLolClientApiInfoV2()
			if err != nil {
				if !errors.Is(lcu.ErrLolProcessNotFound, err) {
					logger.Error("获取lcu info 失败", zap.Error(err))
				}
				continue
			}
			p.initLcuClient(port, token)
			err = p.initGameFlowMonitor(port, token)
			if err != nil {
				logger.Error("游戏流程监视器 err:", err)
			}
			p.lcuActive = false
		}
		time.Sleep(time.Second)
	}

}

func (p Prophet) notifyQuit() error {
	errC := make(chan error, 1)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				errC <- p.ctx.Err()
				return
			case <-interrupt:
				_ = p.Stop()
			}
		}
	}()
	err := <-errC
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (p Prophet) initLcuClient(port int, token string) {
	lcu.InitCli(port, token)
}

func (p Prophet) initGameFlowMonitor(port int, authPwd string) error {
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	rawUrl := fmt.Sprintf("wss://127.0.0.1:%d/", port)
	header := http.Header{}
	authSecret := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("riot:%s", authPwd)))
	header.Set("Authorization", "Basic "+authSecret)
	u, _ := url.Parse(rawUrl)
	logger.Debug(fmt.Sprintf("connect to lcu %s", u.String()))
	c, _, err := dialer.Dial(u.String(), header)
	if err != nil {
		logger.Error("连接到lcu ws 失败", zap.Error(err))
		return err
	}
	p.lcuActive = true
	// if global.IsDevMode() {
	// 	lcu.ChampionSelectStart()
	// }
	defer c.Close()
	_ = c.WriteMessage(websocket.TextMessage, []byte("[5, \"OnJsonApiEvent\"]"))
	for {
		msgType, message, err := c.ReadMessage()
		if err != nil {
			// log.Println("read:", err)
			logger.Error("lol事件监控读取消息失败", zap.Error(err))
			return err
		}
		msg := &wsMsg{}
		if msgType != websocket.TextMessage || len(message) < onJsonApiEventPrefixLen+1 {
			continue
		}
		_ = json.Unmarshal(message[onJsonApiEventPrefixLen:len(message)-1], msg)
		if msg.Uri == gameFlowChangedEvt {
			gameFlow, ok := msg.Data.(string)
			if !ok {
				continue
			}
			logger.Debug("切换状态:" + gameFlow)
			if gameFlow == string(models.GameFlowChampionSelect) {
				log.Println("进入英雄选择阶段,正在计算用户分数")
				sentry.CaptureMessage("进入英雄选择阶段,正在计算用户分数")
				go lcu.ChampionSelectStart()
			}
		}
		// log.Printf("recv: %s", message)
	}
}
