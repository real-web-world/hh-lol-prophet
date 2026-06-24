//go:build windows

package lcu

import (
	"regexp"
	"strconv"

	"github.com/real-web-world/hh-lol-prophet/pkg/windows/process"
)

const (
	lolUxProcessName = "LeagueClientUx.exe"
)

var (
	lolCommandlineReg = regexp.MustCompile(`--remoting-auth-token=(.+?)" ".*?--app-port=(\d+)"`)
)

func GetLolClientApiInfoAdapt() (port int, token string, err error) {
	return GetLolClientApiInfoV3()
}
func GetLolClientApiInfoV3() (port int, token string, err error) {
	cmdline, err := process.GetProcessCommand(lolUxProcessName)
	if err != nil {
		err = ErrLolProcessNotFound
		return
	}
	btsChunk := lolCommandlineReg.FindSubmatch([]byte(cmdline))
	if len(btsChunk) < 3 {
		return port, token, ErrLolProcessNotFound
	}
	token = string(btsChunk[1])
	port, err = strconv.Atoi(string(btsChunk[2]))
	return
}
