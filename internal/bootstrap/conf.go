package bootstrap

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/real-web-world/hh-lol-prophet/internal/config"
	"github.com/real-web-world/hh-lol-prophet/pkg/bdk"
)

const (
	appConfPath      = config.AppConfPath
	clientConfPath   = config.ClientConfPath
	envFileName      = ".env"
	envLocalFileName = ".env.local"
)

func initConf() (*config.Config, error) {
	_ = loadEnv(envFileName, false)
	if bdk.IsFile(envLocalFileName) {
		_ = loadEnv(envLocalFileName, true)
	}
	// prod 模式异步拉取远程配置并落盘; dev 模式直接返回 nil
	remoteConfC := make(chan *config.AppConfig, 1)
	go func() {
		defer func() { _ = recover() }()
		if isEnvModeDev() {
			remoteConfC <- nil
			return
		}
		cfg, _ := getRemoteAppConf()
		if cfg != nil {
			bts, _ := json.Marshal(cfg)
			_ = os.WriteFile(appConfPath, bts, 0664)
		}
		remoteConfC <- cfg
	}()
	// 本地客户端配置初始化(与远程并发)
	clientCfg, err := initClientConf()
	if err != nil {
		return nil, err
	}
	remoteConf := <-remoteConfC
	var appCfg *config.AppConfig
	if remoteConf != nil {
		appCfg = remoteConf
	} else {
		appCfg = loadLocalAppConf()
	}
	return &config.Config{App: appCfg, Client: clientCfg}, nil
}

// loadEnv 解析 .env 文件并设置环境变量。
// override=false 时已存在的环境变量不被覆盖(godotenv.Load 语义);
// override=true 时强制覆盖(godotenv.Overload 语义)。
func loadEnv(path string, override bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if n := len(val); n >= 2 {
			first, last := val[0], val[n-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				val = val[1 : n-1]
			}
		}
		if _, exists := os.LookupEnv(key); exists && !override {
			continue
		}
		_ = os.Setenv(key, val)
	}
	return scanner.Err()
}

func isEnvModeDev() bool {
	return os.Getenv("PROPHET_MODE") == string(config.ModeDebug)
}

func getRemoteAppConf() (*config.AppConfig, error) {
	cli := &http.Client{Timeout: 2 * time.Second}
	resp, err := cli.Get(config.GetRemoteConfApi)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	type buffResp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	res := &buffResp{}
	if err = json.Unmarshal(bts, res); err != nil {
		return nil, err
	}
	cfg := &config.AppConfig{}
	if err = json.Unmarshal(res.Data, cfg); err != nil {
		return nil, err
	}
	if cfg.AppName == "" {
		return nil, errors.New("获取远程配置失败")
	}
	return cfg, nil
}

func loadLocalAppConf() *config.AppConfig {
	appConf := *config.DefaultAppConf
	if bdk.IsFile(appConfPath) {
		bts, err := os.ReadFile(appConfPath)
		if err == nil {
			_ = json.Unmarshal(bts, &appConf)
		}
	}
	applyEnvOverrides(&appConf)
	return &appConf
}

// applyEnvOverrides 按 env tag 用环境变量覆盖对应字段(与 main 的 configor 行为一致)。
func applyEnvOverrides(cfg *config.AppConfig) {
	if v, ok := os.LookupEnv("PROPHET_MODE"); ok && v != "" {
		cfg.Mode = v
	}
	if v, ok := os.LookupEnv("buffApiUrl"); ok && v != "" {
		cfg.BuffApi.Url = v
	}
	if v, ok := os.LookupEnv("logLevel"); ok && v != "" {
		cfg.Log.Level = v
	}
}

func initClientConf() (*config.ClientUserConf, error) {
	if !bdk.IsFile(clientConfPath) {
		return config.DefaultClientUserConf, nil
	}
	cfg := &config.ClientUserConf{}
	cfgFile, err := os.Open(clientConfPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cfgFile.Close() }()
	bts, err := io.ReadAll(cfgFile)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(bts, cfg); err != nil {
		return nil, errors.New("本地配置错误")
	}
	if cfg.ShouldAutoOpenBrowser == nil {
		_ = syscall.Unlink(clientConfPath)
		return config.DefaultClientUserConf, nil
	}
	if err = config.ValidClientUserConf(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
