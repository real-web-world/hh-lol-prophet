package service

import (
	"log/slog"

	"github.com/real-web-world/hh-lol-prophet/internal/app"
	"github.com/real-web-world/hh-lol-prophet/internal/config"
)

type (
	Common struct {
		l   *slog.Logger
		cfg *config.Config
	}
)

func NewCommon(inst *app.App) (*Common, error) {
	s := &Common{
		l:   inst.GetLogger(),
		cfg: inst.GetCfg(),
	}
	return s, nil
}
