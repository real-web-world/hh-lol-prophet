package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/real-web-world/hh-lol-prophet/services/lcu"
)

var (
	cli = http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
)

func main() {
	port := 8098
	lcuPort, lcuToken, err := lcu.GetLolClientApiInfoV2()
	if err != nil {
		log.Fatal(err)
	}
	proxyURL := fmt.Sprintf("https://riot:%s@127.0.0.1:%d", lcuToken, lcuPort)
	go func() {
		ticker := time.NewTicker(time.Second * 3)
		for {
			<-ticker.C
			lcuPort, lcuToken, err := lcu.GetLolClientApiInfoV2()
			if err != nil {
				continue
			}
			updateProxyURL := fmt.Sprintf("https://riot:%s@127.0.0.1:%d", lcuToken, lcuPort)
			if updateProxyURL == proxyURL {
				continue
			}
			proxyURL = updateProxyURL
			log.Println("update lcu:", proxyURL)

		}
	}()
	log.Println("lcu api:", proxyURL)
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		req, _ := http.NewRequest(r.Method, proxyURL+r.URL.Path+"?"+r.URL.RawQuery, nil)
		req.Body = r.Body
		req.Header = r.Header
		resp, err := cli.Do(req)
		if err != nil {
			_, _ = fmt.Fprintf(w, "err : %v", err)
			return
		}
		w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		_, _ = io.Copy(w, resp.Body)
		defer resp.Body.Close()
	}))
	if err != nil {
		log.Fatal(err)
	}
}
