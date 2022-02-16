package logger

import (
	"github.com/getsentry/sentry-go"

	"github.com/real-web-world/hh-lol-prophet/global"
)

func Debug(msg string, keysAndValues ...interface{}) {
	global.Logger.Debugw(msg, keysAndValues...)
}
func Info(msg string, keysAndValues ...interface{}) {
	global.Logger.Infow(msg, keysAndValues...)
}
func Warn(msg string, keysAndValues ...interface{}) {
	go sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelWarning)
		scope.SetExtra("kv", keysAndValues)
		sentry.CaptureMessage(msg)
	})
	global.Logger.Warnw(msg, keysAndValues...)
}
func Error(msg string, keysAndValues ...interface{}) {
	go sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelError)
		scope.SetExtra("kv", keysAndValues)
		sentry.CaptureMessage(msg)
	})
	global.Logger.Errorw(msg, keysAndValues...)
}
