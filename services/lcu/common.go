package lcu

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/real-web-world/hh-lol-prophet/pkg/windows/process"
	"github.com/yusufpapurcu/wmi"
	"go.uber.org/zap"

	"github.com/real-web-world/hh-lol-prophet/pkg/bdk"
	"github.com/real-web-world/hh-lol-prophet/services/logger"
)

const (
	lolUxProcessName = "LeagueClientUx.exe"
)

var (
	lolCommandlineReg     = regexp.MustCompile(`--remoting-auth-token=(.+?)" "--app-port=(\d+)"`)
	ErrLolProcessNotFound = errors.New("未找到lol进程")
)

func GetLolClientApiInfo() (int, string, error) {
	return GetLolClientApiInfoV3()
}
func GetLolClientApiInfoV1(fullPath string) (int, string, error) {
	basePath := filepath.Dir(fullPath)
	f, err := os.Open(basePath + "/lockfile")
	if err != nil {
		return 0, "", errLolClientNotFound
	}
	bts, err := io.ReadAll(f)
	arr := strings.Split(bdk.Bytes2Str(bts), ":")
	if len(arr) != 5 {
		logger.Debug("lol 进程 lockfile内容格式不正确", zap.ByteString("content", bts))
		return 0, "", errLolClientNotFound
	}
	port, err := strconv.Atoi(arr[2])
	if err != nil {
		logger.Debug("lol 进程 lockfile内容 port格式不正确", zap.ByteString("content", bts))
		return 0, "", errLolClientNotFound
	}
	return port, arr[3], nil
}
func GetLolClientApiInfoV2() (int, string, error) {
	type Process struct {
		Commandline string
	}
	var port int
	var token string
	var dst []Process
	err := wmi.Query("select * from Win32_Process  WHERE name='LeagueClientUx.exe'", &dst)
	if err != nil || len(dst) == 0 {
		return port, token, ErrLolProcessNotFound
	}
	btsChunk := lolCommandlineReg.FindSubmatch([]byte(dst[0].Commandline))
	if len(btsChunk) < 3 {
		return port, token, ErrLolProcessNotFound
	}
	token = string(btsChunk[1])
	port, err = strconv.Atoi(string(btsChunk[2]))
	return port, token, err
}
func GetLolClientApiInfoV3() (port int, token string, err error) {
	cmdline, err := process.GetProcessCommand(lolUxProcessName)
	if err != nil {
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
