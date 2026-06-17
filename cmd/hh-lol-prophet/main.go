package main

import (
	"log/slog"
	"os"

	"github.com/real-web-world/hh-lol-prophet/internal/bootstrap"
)

func main() {
	app, err := bootstrap.InitApp()
	if err != nil {
		slog.Error("lol对局先知初始化失败", slog.Any("err", err))
		os.Exit(1)
	}
	if err = app.Run(); err != nil {
		slog.Error("lol对局先知关闭失败", slog.Any("err", err))
	}
}
