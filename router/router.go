package router

import (
	"context"
	"example/timesheet/handler"
	"example/timesheet/redis"
	"example/timesheet/utils"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
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

	r.GET("/root", func(ctx *gin.Context) {

		a := ctx.Query("code")

		file, err := os.ReadFile("credentials.json")
		if err != nil {
			utils.AbortWithError(ctx, err)
		}

		cred, err := google.ConfigFromJSON(file, sheets.SpreadsheetsScope)
		if err != nil {
			utils.AbortWithError(ctx, err)
		}

		tok, err := cred.Exchange(context.TODO(), a)
		if err != nil {
			log.Fatalf("Unable to retrieve token from web: %v", err)
		}

		tokFile := "token.json"
		utils.SaveToken(tokFile, tok)

		ctx.Redirect(http.StatusTemporaryRedirect, "http://localhost:8080")

	})

	r.GET("/", func(ctx *gin.Context) {

		ctx.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Timesheet",
		})
	})

	r.GET("/login", func(ctx *gin.Context) {

		var authURL string

		tokFile := "token.json"
		_, err := utils.TokenFromFile(tokFile)
		if err != nil {

			file, err := os.ReadFile("credentials.json")

			if err != nil {
				utils.AbortWithError(ctx, err)
			}

			config, err := google.ConfigFromJSON(file, sheets.SpreadsheetsScope)
			if err != nil {
				utils.AbortWithError(ctx, err)
			}
			authURL = config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
			ctx.Redirect(http.StatusTemporaryRedirect, authURL)
		}
		ctx.Redirect(http.StatusTemporaryRedirect, "http://localhost:8080")

	})

	{
		sheet := r.Group("/sheet")
		sheetHandler := handler.NewSheetHandler()
		// m := middleware.NewMiddlewares()
		// sheet.Use(m.AuthCheck())
		sheetHandler.Cache = cache
		sheet.POST("/submit", sheetHandler.SubmitSheetId)
		sheet.POST("/new-entry", sheetHandler.AddSheetEntry)
		// sheet.GET("/tab-names", sheetHandler.GetTabNames)
	}

	return r
}
