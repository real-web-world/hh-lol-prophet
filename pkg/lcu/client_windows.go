//go:build windows

package lcu

func (cli Client) fmtClientApiUrl() string {
	return GenerateClientApiUrl(cli.port, cli.authPwd)
}
