package logger

import "go.uber.org/zap/zapcore"

type LogLevelStr = string

const (
	LevelDebugStr LogLevelStr = "debug"
	LevelFatalStr LogLevelStr = "fatal"
	LevelErrorStr LogLevelStr = "error"
	LevelWarnStr  LogLevelStr = "warn"
	LevelInfoStr  LogLevelStr = "info"
)

func Str2ZapLevel(level LogLevelStr) (zapcore.Level, error) {
	return zapcore.ParseLevel(string(level))
}
