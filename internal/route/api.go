package route

import (
	"net/http"

	"github.com/real-web-world/hh-lol-prophet/internal/app"
	api "github.com/real-web-world/hh-lol-prophet/internal/handler"
)

func initAPIRoutes(r *http.ServeMux, inst *app.App) error {
	var err error
	initDevModule(r, inst)
	if err = initV1Module(r, inst); err != nil {
		return err
	}
	return nil
}

func initV1Module(r *http.ServeMux, inst *app.App) error {
	return nil
}

func initDevModule(r *http.ServeMux, _ *app.App) {
	appH := api.NewApp()
	r.HandleFunc("/test", appH.DevHand)
}
