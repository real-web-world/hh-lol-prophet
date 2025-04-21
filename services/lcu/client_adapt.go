//go:build !windows

package lcu

func (cli client) fmtClientApiUrl() string {
	return "http://192.168.3.21:8098"
}
