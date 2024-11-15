package logger

import (
	"github.com/real-web-world/hh-lol-prophet/global"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Debug(msg string, keysAndValues ...interface{}) {
	log(zapcore.DebugLevel, msg, keysAndValues...)
}
func Info(msg string, keysAndValues ...interface{}) {
	log(zapcore.InfoLevel, msg, keysAndValues...)
}
func Warn(msg string, keysAndValues ...interface{}) {
	log(zapcore.WarnLevel, msg, keysAndValues...)
}
func Error(msg string, keysAndValues ...interface{}) {
	log(zapcore.ErrorLevel, msg, keysAndValues...)
}
func log(lvl zapcore.Level, msg string, keysAndValues ...any) {
	userInfo := global.GetUserInfo()
	if userInfo.Summoner != nil {
		summoner := userInfo.Summoner
		keysAndValues = append(keysAndValues,
			zap.String("buff.lol.puuid", summoner.Puuid),
			//zap.String("buff.lol.platformId", summoner.),
			zap.String("buff.lol.gameName", summoner.GameName),
			zap.String("buff.lol.gameTag", summoner.TagLine),
		)
	}
	global.Logger.Logw(lvl, msg, keysAndValues...)
}
