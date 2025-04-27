package lcu

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

var (
	httpCli = &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: time.Second * 30,
	}

	cli *Client
)

type (
	Client struct {
		port    int
		authPwd string
		baseUrl string
	}
)

func InitCli(port int, token string) {
	cli = NewClient(port, token)
}

func NewClient(port int, token string) *Client {
	client := &Client{
		port:    port,
		authPwd: token,
	}
	client.baseUrl = client.fmtClientApiUrl()
	return client
}
func (cli Client) httpGet(url string) ([]byte, error) {
	return cli.req(http.MethodGet, url, nil)
}
func (cli Client) httpPost(url string, body interface{}) ([]byte, error) {
	return cli.req(http.MethodPost, url, body)
}
func (cli Client) httpPatch(url string, body interface{}) ([]byte, error) {
	return cli.req(http.MethodPatch, url, body)
}
func (cli Client) httpDel(url string) ([]byte, error) {
	return cli.req(http.MethodDelete, url, nil)
}
func (cli Client) req(method string, url string, data interface{}) ([]byte, error) {
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
