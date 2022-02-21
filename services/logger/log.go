package logger

import (
	"fmt"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

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
	var errMsg string
	var errVerbose string
	for _, v := range keysAndValues {
		if field, ok := v.(zap.Field); ok && field.Type == zapcore.ErrorType {
			errMsg = field.Interface.(error).Error()
			errVerbose = fmt.Sprintf("%+v", field.Interface.(error))
		}
	}
	go sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelError)
		scope.SetExtra("kv", keysAndValues)
		if errMsg != "" {
			scope.SetExtra("error", errMsg)
			scope.SetExtra("errorVerbose", errVerbose)
		}
		sentry.CaptureMessage(msg)
	})
	global.Logger.Errorw(msg, keysAndValues...)
}
