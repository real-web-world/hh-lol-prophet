//go:build !windows

package lcu

// linux下可以访问windows主机的lcu-agent服务
// 也可以用反向代理访问windows local app->local nginx -> windows nginx -> windows lcu
func (cli Client) fmtClientApiUrl() string {
	return GenerateClientApiUrl(cli.port, cli.authPwd)
}
