package handler

import (
	"encoding/json"
	"net/http"

	apiProj "github.com/real-web-world/hh-lol-prophet"
)

type (
	VersionInfo struct {
		Version   string `json:"version"   example:"1.0.0"`
		Commit    string `json:"commit"    example:"github commit sha256"`
		BuildTime string `json:"buildTime" example:"2006-01-02 15:04:05"`
		BuildUser string `json:"buildUser" example:"buffge"`
	}
)

type (
	AppHandler struct{}
)

func NewApp() *AppHandler {
	return &AppHandler{}
}

func (api *AppHandler) DevHand(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("dev"))
}

func (api *AppHandler) ShowVersion(w http.ResponseWriter, r *http.Request) {
	bts, _ := json.Marshal(VersionInfo{
		Version:   apiProj.APPVersion,
		Commit:    apiProj.Commit,
		BuildTime: apiProj.BuildTime,
		BuildUser: apiProj.BuildUser,
	})
	_, _ = w.Write(bts)
}
