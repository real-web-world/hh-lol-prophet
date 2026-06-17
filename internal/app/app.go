package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/real-web-world/hh-lol-prophet/internal/config"
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
	app.l.Info("lol对局先知启动成功", slog.Duration("耗时", time.Since(app.begin)),
		slog.String("httpAddr", app.cfg.App.HttpServer.Addr),
		slog.String("mode", app.cfg.App.Mode))
	quitCtx, sigCancel := signal.NotifyContext(app.ctx, syscall.SIGINT, syscall.SIGTERM)
	defer sigCancel()
	<-quitCtx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := app.Stop(ctx, context.Cause(quitCtx))
	if err == nil {
		app.l.Info("lol对局先知关闭成功", slog.Any("cause", context.Cause(quitCtx)))
	}
	return err
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
