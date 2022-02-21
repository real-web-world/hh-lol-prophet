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
	sentryGin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/real-web-world/hh-lol-prophet/global"
	ginApp "github.com/real-web-world/hh-lol-prophet/pkg/gin"
	"github.com/real-web-world/hh-lol-prophet/services/lcu"
	"github.com/real-web-world/hh-lol-prophet/services/lcu/models"
	"github.com/real-web-world/hh-lol-prophet/services/logger"
)

type (
	Prophet struct {
		ctx       context.Context
		opts      *options
		httpSrv   *http.Server
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
	options struct {
		debug       bool
		enablePprof bool
		httpAddr    string
	}
)

const (
	onJsonApiEventPrefixLen = len(`[8,"OnJsonApiEvent",`)
	gameFlowChangedEvt      = "/lol-gameflow/v1/gameflow-phase"
)

var (
	defaultOpts = &options{
		debug:       false,
		enablePprof: true,
		httpAddr:    ":4396",
	}
)

func NewProphet(opts ...ApplyOption) *Prophet {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Prophet{
		ctx:    ctx,
		cancel: cancel,
		mu:     &sync.Mutex{},
		opts:   defaultOpts,
	}
	for _, fn := range opts {
		fn(p.opts)
	}
	return p
}
func (p Prophet) Run() error {
	go p.MonitorStart()
	go p.captureStartMessage()
	p.httpStart()
	log.Printf("%s已启动 v%s -- %s", global.AppName, APPVersion, global.WebsiteTitle)
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
			port, token, err := lcu.GetLolClientApiInfo()
			if err != nil {
				if !errors.Is(lcu.ErrLolProcessNotFound, err) {
					logger.Error("获取lcu info 失败", zap.Error(err))
				}
				time.Sleep(time.Second)
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
func (p *Prophet) httpStart() {
	p.httpSrv = p.initGin()
}
func (p Prophet) notifyQuit() error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	g, c := errgroup.WithContext(p.ctx)
	g.Go(func() error {
		err := p.httpSrv.ListenAndServe()
		if err != nil || !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		<-c.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		return p.httpSrv.Shutdown(ctx)
	})
	g.Go(func() error {
		for {
			select {
			case <-p.ctx.Done():
				return p.ctx.Err()
			case <-interrupt:
				_ = p.Stop()
			}
		}
	})
	err := g.Wait()
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

func (p Prophet) captureStartMessage() {
	for i := 0; i < 5; i++ {
		if global.GetUserInfo().CpuID != "" {
			break
		}
		time.Sleep(time.Second * 2)
	}
	sentry.CaptureMessage(global.AppName + "已启动")
}

func (p Prophet) initGin() *http.Server {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.LoggerWithFormatter(logFormatter))
	if p.opts.enablePprof {
		pprof.RouteRegister(engine.Group(""))
	}
	engine.Use(ginApp.PrepareProc)
	engine.Use(sentryGin.New(sentryGin.Options{
		Repanic: true,
		Timeout: 3 * time.Second,
	}))
	if p.opts.debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	RegisterRoutes(engine)
	srv := &http.Server{
		Addr:    p.opts.httpAddr,
		Handler: engine,
	}
	return srv
}

func logFormatter(p gin.LogFormatterParams) string {
	isOptMethod := p.Request.Method == http.MethodOptions
	isSkipLog := p.StatusCode == http.StatusNotFound || isOptMethod
	if isSkipLog {
		return ""
	}
	reqTime := p.TimeStamp.Format("2006-01-02 15:04:05")
	path := p.Request.URL.Path
	method := p.Request.Method
	code := p.StatusCode
	clientIp := p.ClientIP
	errMsg := p.ErrorMessage
	processTime := p.Latency
	return fmt.Sprintf("API: %s %d %s %s %s %v %s\n", reqTime, code, clientIp, path, method, processTime,
		errMsg)
}

type ApplyOption func(o *options)

func WithEnablePprof(enablePprof bool) ApplyOption {
	return func(o *options) {
		o.enablePprof = enablePprof
	}
}
func WithHttpAddr(httpAddr string) ApplyOption {
	return func(o *options) {
		o.httpAddr = httpAddr
	}
}
func WithDebug() ApplyOption {
	return func(o *options) {
		o.debug = true
	}
}
func WithProd() ApplyOption {
	return func(o *options) {
		o.debug = false
	}
}
