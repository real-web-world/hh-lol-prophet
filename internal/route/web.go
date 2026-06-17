package route

import (
	"net/http"

	"github.com/real-web-world/hh-lol-prophet/internal/app"
	"github.com/real-web-world/hh-lol-prophet/internal/handler"
)

func initWebRoutes(r *http.ServeMux, inst *app.App) {
	appH := handler.NewApp()
	r.HandleFunc("/version", appH.ShowVersion)
}
