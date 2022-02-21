package api

import (
	"github.com/gin-gonic/gin"

	ginApp "github.com/real-web-world/hh-lol-prophet/pkg/gin"
)

func DevHand(c *gin.Context) {
	app := ginApp.GetApp(c)
	app.Data(gin.H{
		"buffge": 23456,
	})
}
