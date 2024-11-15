package hh_lol_prophet

import "github.com/real-web-world/hh-lol-prophet/global"

var (
	APPVersion = "0.3.0"
	Commit     = "dev"
	BuildTime  = ""
	BuildUser  = ""
)

func init() {
	global.SetAppInfo(global.AppInfo{
		Version:   APPVersion,
		Commit:    Commit,
		BuildUser: BuildUser,
		BuildTime: BuildTime,
	})
}
