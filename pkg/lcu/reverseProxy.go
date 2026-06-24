package lcu

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
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
		return nil, fmt.Errorf("解析反向代理目标 URL 时出错: %w", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Rewrite = func(preq *httputil.ProxyRequest) {
		preq.Out.URL.Scheme = "https"
		preq.Out.URL.Host = targetURL.Host
		preq.Out.Header["X-Forwarded-For"] = nil
		if targetURL.User != nil {
			pwd, _ := targetURL.User.Password()
			preq.Out.SetBasicAuth(targetURL.User.Username(), pwd)
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
