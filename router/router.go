package router

import (
	"example/timesheet/handler"
	"example/timesheet/redis"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetUpRouter() *gin.Engine {
	router := gin.New()
	router = initRoutes(router)
	return router
}

func initRoutes(r *gin.Engine) *gin.Engine {
	cache := redis.NewCache()
	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "Health is ok")
	})
	r.LoadHTMLGlob("templates/*")
	r.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Timesheet",
		})
	})
	{
		sheet := r.Group("/sheet")
		sheetHandler := handler.NewSheetHandler()
		sheetHandler.Cache = cache
		sheet.POST("/submit", sheetHandler.SubmitSheetId)
	}

	return r
}
