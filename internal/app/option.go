package app

import (
	"log/slog"
	"net/http"

	"github.com/real-web-world/hh-lol-prophet/internal/config"
)

type Option func(*App)

func WithConfig(cfg *config.Config) Option {
	return func(app *App) {
		app.cfg = cfg
	}
}

func WithLogger(l *slog.Logger) Option {
	return func(app *App) {
		app.l = l
	}
}

func WithServer(srv *http.Server) Option {
	return func(app *App) {
		app.srv = srv
	}
}
