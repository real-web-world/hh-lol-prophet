package ginApp

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/real-web-world/hh-lol-prophet/global"
)

func LogFormatter(p gin.LogFormatterParams) string {
	if !global.IsDevMode() {
		return ""
	}
	isOptMethod := p.Request.Method == http.MethodOptions
	isSkipLog := p.StatusCode == http.StatusNotFound || isOptMethod
	if isSkipLog {
		return ""
	}
	reqTime := p.TimeStamp.Format("2006-01-02 15:04:05")
	path := p.Request.URL.Path
	method := p.Request.Method
	code := p.StatusCode
	clientIp := p.ClientIP
	errMsg := p.ErrorMessage
	processTime := p.Latency
	return fmt.Sprintf("API: %s %d %s %s %s %v %s\n", reqTime, code, clientIp, path, method, processTime,
		errMsg)
}
