package bootstrap

import (
	"io"
	"log/slog"
	"os"

	"github.com/real-web-world/hh-lol-prophet/internal/config"
)

func initSlogJSON(level slog.Level, writer io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     &level,
		AddSource: false,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				t := a.Value.Time()
				return slog.String(slog.TimeKey, t.Format("2006-01-02 15:04:05.000"))
			}
			if a.Value.Kind() == slog.KindDuration {
				d := a.Value.Duration()
				return slog.String(a.Key, d.String())
			}
			return a
		},
	}
	jsonHandler := slog.NewJSONHandler(writer, opts)
	l := slog.New(jsonHandler)
	slog.SetDefault(l)
	return l
}

func initLog(cfg *config.Config) (*slog.Logger, error) {
	var logWriter io.WriteCloser = os.Stdout
	logLevel := slog.LevelDebug
	if cfg.IsProd() {
		logLevel = slog.LevelInfo
	}
	l := initSlogJSON(logLevel, logWriter)
	return l, nil
}
