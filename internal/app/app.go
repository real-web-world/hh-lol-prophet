package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	appProj "github.com/real-web-world/hh-lol-prophet"
	"github.com/real-web-world/hh-lol-prophet/internal/config"
	"github.com/real-web-world/hh-lol-prophet/internal/prophet"
	"github.com/real-web-world/hh-lol-prophet/pkg/bdk"
)

type (
	App struct {
		mu     *sync.Mutex
		ctx    context.Context
		cancel context.CancelCauseFunc
		begin  time.Time
		cfg    *config.Config
		srv    *http.Server
		l      *slog.Logger
		p      *prophet.Prophet
		Third  *ThirdParty
	}
)

func New(ctx context.Context, opts ...Option) *App {
	ctx, cancel := context.WithCancelCause(ctx)
	app := &App{
		mu:     &sync.Mutex{},
		ctx:    ctx,
		cancel: cancel,
		begin:  time.Now(),
	}
	for _, opt := range opts {
		opt(app)
	}
	return app
}

func (app *App) Run() error {
	go app.StartHttpSrv()
	app.l.Debug("lol对局先知启动成功", slog.Duration("耗时", time.Since(app.begin)),
		slog.String("httpAddr", app.cfg.App.HttpServer.Addr),
		slog.String("mode", app.cfg.App.Mode))
	log.Printf("%s已启动 v%s -- %s", app.cfg.App.AppName, appProj.APPVersion, app.cfg.App.WebsiteTitle)
	quitCtx, sigCancel := signal.NotifyContext(app.ctx, syscall.SIGINT, syscall.SIGTERM)
	defer sigCancel()
	<-quitCtx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := app.Stop(ctx, context.Cause(quitCtx))
	if err == nil {
		app.l.Debug("lol对局先知关闭成功", slog.Any("cause", context.Cause(quitCtx)))
	}
	return err
}

func (app *App) checkUpdate() error {
	if app.cfg.IsDebug() {
		return nil
	}
	var binNewFullPath string
	updateInfo, err := app.Third.BuffApi.GetCurrVersion()
	if err != nil || updateInfo.VersionTag == "" || updateInfo.DownloadUrl == "" {
		return nil
	}
	version := strings.TrimLeft(updateInfo.VersionTag, "v")
	if bdk.CompareVersion(version, appProj.APPVersion) <= 0 {
		// log.Println("已是最新无需下载")
		return nil
	}
	log.Println("检测到更新,两秒后将更新或按回车立即更新")
	updateTimeoutC := time.After(time.Second * 2)
	done := make(chan struct{})
	go func() {
		_, _ = fmt.Scanln(new(""))
		done <- struct{}{}
	}()
	// download new.exe
	{
		resp, err := http.Get(updateInfo.DownloadUrl)
		if err != nil {
			log.Println("下载最新二进制失败")
			return err
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		binNewPath, err := os.Executable()
		if err != nil {
			return err
		}
		dirPath, err := filepath.Abs(filepath.Dir(binNewPath))
		if err != nil {
			return err
		}
		binNewFullPath = filepath.Join(dirPath, config.ProcNewName)
		binNewFile, err := os.OpenFile(binNewFullPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
		if err != nil {
			return err
		}
		_, err = io.Copy(binNewFile, resp.Body)
		if err != nil {
			_ = binNewFile.Close()
			return errors.New("下载更新文件失败")
		}
		_ = binNewFile.Close()
	}
	select {
	case <-done:
	case <-updateTimeoutC:
	}
	// 输出信息 即将更新
	// 启动ttt_new.exe
	{
		cmd := exec.Command("cmd.exe", "/C", "start", binNewFullPath, "-u", "true")
		if err := cmd.Run(); err != nil {
			log.Println("启动更新进程失败:", err)
			return err
		}
		os.Exit(0)
	}
	return nil
}

func (app *App) StartHttpSrv() {
	if err := app.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		app.l.Error("lol对局先知 http服务启动失败", slog.Any("err", err))
		app.cancel(err)
	}
}

func (app *App) SrvShutdown(ctx context.Context) error {
	if err := app.srv.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

func (app *App) Stop(ctx context.Context, cause error) error {
	app.cancel(cause)
	if err := app.SrvShutdown(ctx); err != nil {
		app.l.Debug("http hh-lol-prophet shutdown err", slog.Any("err", err))
	}
	return nil
}

func (app *App) IsDebug() bool {
	return app.cfg.IsDebug()
}

func (app *App) GetCfg() *config.Config {
	return app.cfg
}

func (app *App) GetHttpSrv() *http.Server {
	return app.srv
}

func (app *App) GetLogger() *slog.Logger {
	return app.l
}

func (app *App) SetCfg(cfg *config.Config) {
	app.cfg = cfg
}

func (app *App) SetHttpSrv(srv *http.Server) {
	app.srv = srv
}

func (app *App) SetLogger(l *slog.Logger) {
	app.l = l
}
