package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/real-web-world/hh-lol-prophet/internal/app"
	"github.com/real-web-world/hh-lol-prophet/internal/config"
	"github.com/real-web-world/hh-lol-prophet/internal/prophet"
	"github.com/real-web-world/hh-lol-prophet/internal/route"
)

func InitApp() (*app.App, error) {
	inst := app.New(context.TODO())
	cfg, err := initConf()
	if err != nil {
		return nil, err
	}
	inst.SetCfg(cfg)
	l, err := initLog(cfg)
	if err != nil {
		return nil, err
	}
	inst.SetLogger(l)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err = initLib(ctx); err != nil {
		return nil, err
	}
	if err = initInfra(ctx, inst); err != nil {
		return nil, err
	}
	if err = initThird(ctx, inst); err != nil {
		return nil, err
	}
	p := prophet.NewProphet(l, cfg.App, cfg.Client)
	inst.SetProphet(p)
	httpSrv, err := initHttpSrv(inst, &cfg.App.HttpServer)
	if err != nil {
		return nil, err
	}
	inst.SetHttpSrv(httpSrv)
	return inst, err
}

func initThird(ctx context.Context, inst *app.App) error {
	third := &app.ThirdParty{}
	if err := third.Init(ctx, inst.GetCfg()); err != nil {
		return err
	}
	inst.Third = third
	return nil
}

func initHttpSrv(inst *app.App, cfg *config.HttpServerConf) (*http.Server, error) {
	r, err := initEngine(inst)
	if err != nil {
		return nil, err
	}
	srv := &http.Server{
		Addr:           cfg.Addr,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 5 * 1 << 10,
	}
	return srv, nil
}

func globalMiddleware(l *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				l.Debug("http panic", slog.Any("err", err))
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"code":1,msg":"服务器错误"}`))
			}
		}()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func initEngine(inst *app.App) (http.Handler, error) {
	mux := http.NewServeMux()
	r := globalMiddleware(inst.GetLogger(), mux)
	initMiddleware(r, inst)
	if err := route.RouterSetup(mux, inst); err != nil {
		return nil, err
	}
	return r, nil
}

func initMiddleware(r http.Handler, inst *app.App) {
}
