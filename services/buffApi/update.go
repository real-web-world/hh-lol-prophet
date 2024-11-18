package buffApi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/real-web-world/hh-lol-prophet/conf"
)

const (
	CodeOk = 0
)

var (
	client  *http.Client
	baseUrl string
)

type (
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

func Init(url string, timeoutSec int) {
	client = &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
	}
	baseUrl = url
}
func req(reqPath string, body []byte) ([]byte, error) {
	resp, err := client.Post(baseUrl+reqPath, "application/json", bytes.NewBuffer(body))
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
func GetClientConf() (*conf.CalcScoreConf, error) {
	data, err := req("/lol/client/getConf", nil)
	if err != nil {
		return nil, err
	}
	scoreConf := &conf.CalcScoreConf{}
	err = json.Unmarshal(data, scoreConf)
	if err != nil {
		return nil, err
	}
	return scoreConf, nil
}
func GetCurrVersion() (*CurrVersion, error) {
	data, err := req("/lol/getCurrVersion", nil)
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
