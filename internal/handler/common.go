package handler

import (
	"github.com/real-web-world/hh-lol-prophet/internal/app"
	"github.com/real-web-world/hh-lol-prophet/internal/service"
)

type (
	Common struct {
		s *service.Common
	}
)

func NewCommon(inst *app.App) (*Common, error) {
	s, err := service.NewCommon(inst)
	if err != nil {
		return nil, err
	}
	h := &Common{s: s}
	return h, nil
}
