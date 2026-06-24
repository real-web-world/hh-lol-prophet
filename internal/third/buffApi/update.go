package buffApi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/real-web-world/hh-lol-prophet/internal/config"
)

const (
	CodeOk = 0
)

type (
	Client struct {
		cli     *http.Client
		baseUrl string
	}
	Response struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	CurrVersion struct {
		DownloadUrl    string `json:"downloadUrl"`
		VersionTag     string `json:"versionTag"`
		ZipDownloadUrl string `json:"zipDownloadUrl"`
	}
)

func NewClient(baseUrl string, timeoutSec int) *Client {
	cli := &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
	}
	return &Client{
		cli:     cli,
		baseUrl: baseUrl,
	}
}

func (api Client) req(reqPath string, body []byte) ([]byte, error) {
	resp, err := api.cli.Post(api.baseUrl+reqPath, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	bts, _ := io.ReadAll(resp.Body)
	apiResp := &Response{}
	_ = json.Unmarshal(bts, apiResp)
	if apiResp.Code != CodeOk {
		return nil, errors.New(apiResp.Msg)
	}
	return apiResp.Data, nil
}

func (api Client) GetClientConf() (*config.CalcScoreConf, error) {
	data, err := api.req("/v1/lol/client/getConf", nil)
	if err != nil {
		return nil, err
	}
	scoreConf := &config.CalcScoreConf{}
	err = json.Unmarshal(data, scoreConf)
	if err != nil {
		return nil, err
	}
	return scoreConf, nil
}

func (api Client) GetCurrVersion() (*CurrVersion, error) {
	data, err := api.req("/v1/lol/getCurrVersion", nil)
	if err != nil {
		return nil, err
	}
	versionInfo := &CurrVersion{}
	err = json.Unmarshal(data, versionInfo)
	if err != nil {
		return nil, err
	}
	return versionInfo, nil
}
