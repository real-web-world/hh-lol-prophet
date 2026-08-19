package lcu

import (
	"crypto/tls"
	"fmt"
	"log/slog"
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
	proxy := &httputil.ReverseProxy{
		Rewrite: func(preq *httputil.ProxyRequest) {
			preq.SetURL(targetURL)
			preq.Out.Header.Del("X-Forwarded-For")
			if targetURL.User != nil {
				pwd, _ := targetURL.User.Password()
				preq.Out.SetBasicAuth(targetURL.User.Username(), pwd)
			}
			slog.Debug("debug rp", slog.Any("preq", preq.Out.URL.RequestURI()))
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "proxy error: "+err.Error(), http.StatusBadGateway)
		},
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
