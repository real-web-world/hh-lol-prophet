package app

import (
	"context"

	"github.com/real-web-world/hh-lol-prophet/internal/config"
	"github.com/real-web-world/hh-lol-prophet/internal/third/buffApi"
)

type (
	ThirdParty struct {
		BuffApi *buffApi.Client
	}
)

func (t *ThirdParty) Init(_ context.Context, cfg *config.Config) error {
	apiCfg := cfg.App.BuffApi
	t.BuffApi = buffApi.NewClient(apiCfg.Url, apiCfg.Timeout)
	return nil
}
