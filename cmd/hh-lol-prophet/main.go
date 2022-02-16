package main

import (
	"fmt"
	"log"

	hh_lol_prophet "github.com/real-web-world/hh-lol-prophet"
	"github.com/real-web-world/hh-lol-prophet/bootstrap"
	"github.com/real-web-world/hh-lol-prophet/global"
)

func main() {
	err := bootstrap.InitApp()
	defer global.Cleanup()
	if err != nil {
		panic(fmt.Sprintf("初始化应用失败:%v\n", err))
	}
	app := hh_lol_prophet.NewProphet()
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
