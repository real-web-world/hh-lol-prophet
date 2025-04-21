//go:build windows

package lcu

import (
	"fmt"
)

func (cli client) fmtClientApiUrl() string {
	return fmt.Sprintf("https://riot:%s@127.0.0.1:%d", cli.authPwd, cli.port)
}
