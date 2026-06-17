package route

import (
	"net/http"

	"github.com/real-web-world/hh-lol-prophet/internal/app"
)

func RouterSetup(mux *http.ServeMux, inst *app.App) error {
	var err error
	initWebRoutes(mux, inst)
	if err = initAPIRoutes(mux, inst); err != nil {
		return err
	}
	return nil
}
