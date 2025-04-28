package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/real-web-world/hh-lol-prophet/global"
)

func Debug(msg string, keysAndValues ...any) {
	log(zapcore.DebugLevel, msg, keysAndValues...)
}
func Info(msg string, keysAndValues ...any) {
	log(zapcore.InfoLevel, msg, keysAndValues...)
}
func Warn(msg string, keysAndValues ...any) {
	log(zapcore.WarnLevel, msg, keysAndValues...)
}
func Error(msg string, keysAndValues ...any) {
	log(zapcore.ErrorLevel, msg, keysAndValues...)
}
func log(lvl zapcore.Level, msg string, keysAndValues ...any) {
	userInfo := global.GetUserInfo()
	if userInfo.Summoner != nil {
		summoner := userInfo.Summoner
		keysAndValues = append(keysAndValues,
			zap.String("buff.lol.puuid", summoner.Puuid),
			zap.String("buff.lol.platformId", summoner.PlatformId),
			zap.String("buff.lol.gameName", summoner.GameName),
			zap.String("buff.lol.gameTag", summoner.GameTag),
			zap.String("buff.lol.level", summoner.Lol.Level),
		)
	}
	global.Logger.Logw(lvl, msg, keysAndValues...)
}
