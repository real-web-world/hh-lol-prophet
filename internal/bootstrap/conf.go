package bootstrap

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/real-web-world/hh-lol-prophet/internal/config"
	"github.com/real-web-world/hh-lol-prophet/pkg/bdk"
)

const (
	dataPath = "./prophet.data"
)

func initConf() (*config.Config, error) {
	appCfg := initAppConf()
	clientCfg := initClientConf()
	cfg := &config.Config{
		App:    appCfg,
		Client: clientCfg,
	}
	return cfg, nil
}

func initAppConf() *config.AppConfig {
	appConf := config.DefaultAppConf
	configPath := filepath.Join(defaultConfPath, configFile)
	if !bdk.IsFile(configPath) {
		return appConf
	}
	cfgFile, err := os.Open(configPath)
	if err != nil {
		return appConf
	}
	defer func() {
		_ = cfgFile.Close()
	}()
	bts, _ := io.ReadAll(cfgFile)
	_ = json.Unmarshal(bts, appConf)
	if appConf.HttpServer.Addr == "" {
		return config.DefaultAppConf
	}
	return appConf
}

func initClientConf() *config.ClientUserConf {
	if !bdk.IsFile(dataPath) {
		return config.DefaultClientUserConf
	}
	cfg := &config.ClientUserConf{}
	cfgFile, err := os.Open(dataPath)
	if err != nil {
		return config.DefaultClientUserConf
	}
	defer func() {
		_ = cfgFile.Close()
	}()
	bts, _ := io.ReadAll(cfgFile)
	_ = json.Unmarshal(bts, cfg)
	if cfg.ShouldAutoOpenBrowser == nil {
		_ = syscall.Unlink(dataPath)
		return config.DefaultClientUserConf
	}
	return cfg
}
