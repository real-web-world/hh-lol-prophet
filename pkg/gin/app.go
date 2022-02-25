package ginApp

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"

	"github.com/real-web-world/hh-lol-prophet/global"
	"github.com/real-web-world/hh-lol-prophet/pkg/dto/retcode"
	"github.com/real-web-world/hh-lol-prophet/pkg/fastcurd"
)

const (
	// head field
	HeadLocale      = "locale"
	HeadToken       = "token"
	HeadUserAgent   = "User-Agent"
	HeadContentType = "Content-Type"
	// ctx key
	KeyUID            = "uid"
	KeyTeam           = "team"
	KeyTrans          = "trans"
	KeyApp            = "app"
	KeyAuthUser       = "authUser"
	KeyInitOnce       = "initOnce"
	KeyProcBeginTime  = "procBeginTime"
	KeyNotSaveResp    = "notSaveResp"
	KeyNotTrace       = "notTrace"
	KeyResp           = "resp"
	KeyReqID          = "reqID"
	KeyApiCacheKey    = "apiCacheKey"
	KeyApiCacheExpire = "apiCacheExpire"
	KeyRoles          = "roles"
	KeyStatusCode     = "statusCode"
	KeyProcTime       = "procTime"
)

var (
	respBadReq       = &fastcurd.RetJSON{Code: retcode.BadReq, Msg: "bad request"}
	respNoChange     = &fastcurd.RetJSON{Code: retcode.DefaultError, Msg: "no change"}
	respNoAuth       = &fastcurd.RetJSON{Code: retcode.NoAuth, Msg: "no auth"}
	respNoLogin      = &fastcurd.RetJSON{Code: retcode.NoLogin, Msg: "please login"}
	respReqFrequency = &fastcurd.RetJSON{Code: retcode.RateLimitError, Msg: "high rate req~"}
)

type (
	App struct {
		C  *gin.Context
		mu sync.Mutex
	}
	ServerInfo struct {
		Timestamp int64 `json:"timestamp"`
	}
)

func GetApp(c *gin.Context) *App {
	initOnce, ok := c.Get(KeyInitOnce)
	if !ok {
		panic("ctx must set initOnce")
	}
	initOnce.(*sync.Once).Do(func() {
		c.Set(KeyApp, newApp(c))
	})
	app, _ := c.Get(KeyApp)
	return app.(*App)
}
func newApp(c *gin.Context) *App {
	return &App{
		C: c,
	}
}

// finally resp fn
func (app *App) Response(code int, json *fastcurd.RetJSON) {
	app.SetCtxRespVal(json)
	procBeginTime := app.GetProcBeginTime()
	reqID := app.GetReqID()
	var procTime string
	if procBeginTime != nil {
		procTime = time.Since(*procBeginTime).String()
	}
	json.Extra = &fastcurd.RespJsonExtra{
		ProcTime: procTime,
		ReqID:    reqID,
	}
	app.SetStatusCode(code)
	app.SetProcTime(procTime)
	app.C.JSON(code, json)
	app.C.Abort()
}

// resp helper
func (app *App) Ok(msg string, data ...interface{}) {
	var actData interface{} = nil
	if len(data) == 1 {
		actData = data[0]
	}
	app.JSON(&fastcurd.RetJSON{Code: retcode.Ok, Msg: msg, Data: actData})
}
func (app *App) Data(data interface{}) {
	app.RetData(data)
}
func (app *App) ServerError(err error) {
	json := &fastcurd.RetJSON{Code: retcode.ServerError, Msg: err.Error()}
	app.Response(http.StatusInternalServerError, json)
}
func (app *App) ServerBad() {
	json := &fastcurd.RetJSON{Code: retcode.ServerError, Msg: "服务器开小差了~"}
	app.Response(http.StatusInternalServerError, json)
}
func (app *App) RetData(data interface{}, msgParam ...string) {
	msg := ""
	if len(msgParam) == 1 {
		msg = msgParam[0]
	}
	app.Ok(msg, data)
}
func (app *App) JSON(json *fastcurd.RetJSON) {
	app.Response(http.StatusOK, json)
}
func (app *App) XML(xml interface{}) {
	procBeginTime := app.GetProcBeginTime()
	var procTime string
	if procBeginTime != nil {
		procTime = time.Since(*procBeginTime).String()
	}
	code := http.StatusOK
	app.SetStatusCode(code)
	app.SetProcTime(procTime)
	app.C.XML(code, xml)
	app.C.Abort()
}
func (app *App) SendList(list interface{}, count int) {
	app.Response(http.StatusOK, &fastcurd.RetJSON{
		Code:  retcode.Ok,
		Data:  list,
		Count: &count,
	})
}
func (app *App) BadReq() {
	app.Response(http.StatusBadRequest, respBadReq)
}
func (app *App) String(html string) {
	app.C.String(http.StatusOK, html)
}
func (app *App) ValidError(err error) {
	json := &fastcurd.RetJSON{}
	switch actErr := err.(type) {
	case validator.ValidationErrors:
		json.Code = retcode.ValidError
		json.Msg = actErr[0].Error()
	default:
		if err.Error() == "EOF" {
			json.Code = retcode.ValidError
			json.Msg = "request param is required"
		} else {
			json.Code = retcode.DefaultError
			json.Msg = actErr.Error()
		}
	}
	app.Response(http.StatusBadRequest, json)
}
func (app *App) NoChange() {
	app.JSON(respNoChange)
}
func (app *App) NoAuth() {
	app.Response(http.StatusUnauthorized, respNoAuth)
}
func (app *App) NoLogin() {
	app.Response(http.StatusUnauthorized, respNoLogin)
}
func (app *App) ErrorMsg(msg string) {
	json := &fastcurd.RetJSON{Code: retcode.DefaultError, Msg: msg}
	app.Response(http.StatusOK, json)
}
func (app *App) CommonError(err error) {
	app.ErrorMsg(err.Error())
}
func (app *App) RateLimitError() {
	app.Response(http.StatusBadRequest, respReqFrequency)
}
func (app *App) Success() {
	json := &fastcurd.RetJSON{Code: retcode.Ok}
	app.Response(http.StatusOK, json)
}
func (app *App) SendAffectRows(affectRows int) {
	app.Data(gin.H{
		"affectRows": affectRows,
	})
}

func (app *App) GetFullReqURL() string {
	schema := "http://"
	req := app.C.Request
	if req.TLS != nil {
		schema = "https://"
	}
	return schema + req.Host + req.RequestURI
}

// head field helper
func (app *App) GetLocale() string {
	return app.C.Request.Header.Get(HeadLocale)
}
func (app *App) GetUserAgent() string {
	return app.GetUA()
}
func (app *App) GetToken() string {
	return app.C.Request.Header.Get(HeadToken)
}
func (app *App) GetUA() string {
	return app.C.GetHeader(HeadUserAgent)
}
func (app *App) GetContentType() string {
	return app.C.GetHeader(HeadContentType)
}
func (app *App) GetNotSaveResp() *bool {
	b, ok := app.C.Get(KeyNotSaveResp)
	if !ok {
		return nil
	}
	if t, ok := b.(bool); ok {
		return &t
	}
	return nil
}
func (app *App) SetNotSaveResp() {
	app.C.Set(KeyNotSaveResp, true)
}
func (app *App) GetNotTrace() *bool {
	b, ok := app.C.Get(KeyNotTrace)
	if !ok {
		return nil
	}
	if t, ok := b.(bool); ok {
		return &t
	}
	return nil
}
func (app *App) SetNotTrace() {
	app.C.Set(KeyNotTrace, true)
}
func (app *App) IsShouldSaveResp() bool {
	t := app.GetNotSaveResp()
	return t == nil || *t
}

func (app *App) GetCtxRespVal() *fastcurd.RetJSON {
	if json, ok := app.C.Get(KeyResp); ok {
		return json.(*fastcurd.RetJSON)
	}
	return nil
}
func (app *App) SetCtxRespVal(json *fastcurd.RetJSON) {
	app.C.Set(KeyResp, json)
}
func (app *App) GetProcBeginTime() *time.Time {
	if procBeginTime, ok := app.C.Get(KeyProcBeginTime); ok {
		return procBeginTime.(*time.Time)
	}
	return nil
}
func (app *App) SetProcBeginTime(beginTime *time.Time) {
	app.C.Set(KeyProcBeginTime, beginTime)
}
func (app *App) GetReqID() string {
	if reqID, ok := app.C.Get(KeyReqID); ok {
		return reqID.(string)
	}
	return ""
}
func (app *App) SetReqID(reqID string) {
	app.C.Set(KeyReqID, reqID)
}
func (app *App) GetStatusCode() int {
	if code, ok := app.C.Get(KeyStatusCode); ok {
		return code.(int)
	}
	return 0
}
func (app *App) SetStatusCode(code int) {
	app.C.Set(KeyStatusCode, code)
}
func (app *App) GetProcTime() string {
	if procTime, ok := app.C.Get(KeyProcTime); ok {
		return procTime.(string)
	}
	return ""
}
func (app *App) SetProcTime(procTime string) {
	app.C.Set(KeyProcTime, procTime)
}
func (app *App) GetApiCacheKey() *string {
	if key, ok := app.C.Get(KeyApiCacheKey); ok {
		fmtKey := key.(string)
		return &fmtKey
	}
	return nil
}
func (app *App) SetApiCacheKey(key string) {
	app.C.Set(KeyApiCacheKey, key)
}
func (app *App) GetApiCacheExpire() *time.Duration {
	if ex, ok := app.C.Get(KeyApiCacheExpire); ok {
		return ex.(*time.Duration)
	}
	return nil
}
func (app *App) SetApiCacheExpire(d *time.Duration) {
	app.C.Set(KeyApiCacheExpire, d)
}

// middleware
func PrepareProc(c *gin.Context) {
	now := time.Now()
	c.Set(KeyInitOnce, &sync.Once{})
	c.Set(KeyProcBeginTime, &now)
	c.Next()
}
func ErrHandler(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			var actErr error
			if global.IsDevMode() {
				log.Println("发生异常: ", err)
				debug.PrintStack()
			} else {
				sentry.CaptureException(errors.New(fmt.Sprintf("%v", err)))
			}
			app := GetApp(c)
			switch err := err.(type) {
			case error:
				actErr = err
			case string:
				errMsg := err
				actErr = errors.New(errMsg)
			default:
				actErr = errors.New("server exception")
			}
			app.ServerError(actErr)
			return
		}
	}()
	c.Next()
}
func Cors() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodConnect,
			http.MethodOptions,
			http.MethodTrace,
		},
		AllowHeaders:     []string{"content-type", "x-requested-with", "token", "locale"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
