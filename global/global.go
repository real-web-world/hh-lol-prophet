package global

import (
	"log"
	"sync"

	"go.uber.org/zap"

	"github.com/real-web-world/hh-lol-prophet/conf"
)

type (
	UserInfo struct {
		IP    string `json:"ip"`
		Mac   string `json:"mac"`
		CpuID string `json:"cpuID"`
	}
)

const (
	LogWriterCleanupKey = "logWriter"
)

var (
	userInfo    = UserInfo{}
	scoreConfMu = sync.Mutex{}
	Conf        = &conf.AppConf{}
	Logger      *zap.SugaredLogger
	Cleanups    = make(map[string]func() error)
)

func SetUserInfo(info UserInfo) {
	userInfo = info
}
func GetUserInfo() UserInfo {
	return userInfo
}
func Cleanup() {
	for name, cleanup := range Cleanups {
		if err := cleanup(); err != nil {
			log.Printf("%s cleanup err:%v\n", name, err)
		}
	}
	if fn, ok := Cleanups[LogWriterCleanupKey]; ok {
		_ = fn()
	}
}
func IsDevMode() bool {
	return GetEnv() == conf.ModeDebug
}
func GetEnv() conf.Mode {
	return Conf.Mode
}
func GetScoreConf() conf.CalcScoreConf {
	scoreConfMu.Lock()
	defer scoreConfMu.Unlock()
	return Conf.CalcScore
}
func SetScoreConf(scoreConf conf.CalcScoreConf) {
	scoreConfMu.Lock()
	Conf.CalcScore = scoreConf
	scoreConfMu.Unlock()
	return
}
