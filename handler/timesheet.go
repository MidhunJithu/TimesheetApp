package handler

import (
	"encoding/json"
	"example/timesheet/models"
	"example/timesheet/utils"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SheetSrv struct {
	Cache *models.Cache
}

func NewSheetHandler() *SheetSrv {
	return &SheetSrv{}
}

func (*SheetSrv) SubmitSheetId(ctx *gin.Context) {
	srv, err := GetSheetClient(ctx)
	if err != nil {
		return
	}
	body, err := io.ReadAll(ctx.Request.Body)
	defer ctx.Request.Body.Close()
	if err != nil {
		log.Fatalf("Unable to read request body: %v", err)
	}
	jsonMap := make(map[string]interface{})
	err = json.Unmarshal(body, &jsonMap)
	if err != nil {
		log.Fatalf("Unable to marshall request body: %v", err)
	}
	sheetId, ok := jsonMap["sheetId"]
	if !ok {
		log.Fatalf("No sheet id present: %v", err)
	}
	sheetName, ok := jsonMap["sheetName"]
	if !ok {
		log.Fatalf("No sheet id present: %v", err)
	}
	resp, err := srv.Spreadsheets.Values.Get(sheetId.(string), sheetName.(string)).Do()
	if err != nil {
		utils.AbortWithError(ctx, err)
		return
	}
	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
	} else {
		for _, row := range resp.Values {
			fmt.Printf("%s\n", row)
		}
	}
	ctx.JSON(http.StatusOK, gin.H{
		"data": "ok",
	})
}

func GetSheetClient(ctx *gin.Context) (*sheets.Service, error) {
	file, err := os.ReadFile("credentials.json")
	if err != nil {
		utils.AbortWithError(ctx, err)
		return nil, err
	}
	cred, err := google.ConfigFromJSON(file, sheets.SpreadsheetsScope)
	if err != nil {
		utils.AbortWithError(ctx, err)
		return nil, err
	}
	client := utils.GetClient(cred)

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	return srv, nil
}
