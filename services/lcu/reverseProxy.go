package lcu

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/pkg/errors"
)

type (
	RP struct {
		proxy *httputil.ReverseProxy
		token string
		port  int
	}
)

func NewRP(port int, token string) (*RP, error) {
	targetURL, err := url.Parse(GenerateClientApiUrl(port, token))
	if err != nil {
		return nil, errors.Errorf("解析反向代理目标 URL 时出错: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		req.Host = targetURL.Host
		req.Header["X-Forwarded-For"] = nil
		if targetURL.User != nil {
			pwd, _ := targetURL.User.Password()
			req.SetBasicAuth(targetURL.User.Username(), pwd)
		}
	}
	proxy.Transport = &http.Transport{
		ForceAttemptHTTP2: true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	rp := &RP{
		proxy: proxy,
		token: token,
		port:  port,
	}
	return rp, nil
}
func (rp RP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rp.proxy.ServeHTTP(w, r)
}
