package buffApi

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/real-web-world/hh-lol-prophet/conf"
)

var (
	client  *http.Client
	baseUrl string
)

func Init(url string, timeoutSec int) {
	client = &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
	}
	baseUrl = url
}
func GetCurrConf() (*conf.CalcScoreConf, error) {
	resp, err := client.Get(baseUrl + "/api/currConf.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bts, _ := io.ReadAll(resp.Body)
	scoreConf := &conf.CalcScoreConf{}
	err = json.Unmarshal(bts, scoreConf)
	if err != nil {
		return nil, err
	}
	return scoreConf, nil
}
