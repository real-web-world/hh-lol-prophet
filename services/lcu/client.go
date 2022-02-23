package lcu

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	lolProcessName = "LeagueClientUxRender.exe"
)

var (
	httpCli = &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	errLolClientNotFound = errors.New("未找到lol进程")
	cli                  *client
)

type (
	client struct {
		port    int
		authPwd string
		baseUrl string
	}
)

func InitCli(port int, token string) {
	cli = NewClient(port, token)
}
func (cli client) fmtClientApiUrl() string {
	return fmt.Sprintf("https://riot:%s@127.0.0.1:%d", cli.authPwd, cli.port)
}
func NewClient(port int, token string) *client {
	cli := &client{
		port:    port,
		authPwd: token,
	}
	cli.baseUrl = cli.fmtClientApiUrl()
	return cli
}
func (cli client) httpGet(url string) ([]byte, error) {
	return cli.req(http.MethodGet, url, nil)
}
func (cli client) httpPost(url string, body interface{}) ([]byte, error) {
	return cli.req(http.MethodPost, url, body)
}
func (cli client) httpPatch(url string, body interface{}) ([]byte, error) {
	return cli.req(http.MethodPatch, url, body)
}
func (cli client) httpDel(url string) ([]byte, error) {
	return cli.req(http.MethodDelete, url, nil)
}
func (cli client) req(method string, url string, data interface{}) ([]byte, error) {
	var body io.Reader
	if data != nil {
		bts, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(bts)
	}
	req, _ := http.NewRequest(method, cli.baseUrl+url, body)
	if req.Body != nil {
		req.Header.Add("ContentType", "application/json")
	}
	resp, err := httpCli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
