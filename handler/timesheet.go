package handler

import (
	"encoding/json"
	"example/timesheet/models"
	"example/timesheet/utils"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

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

func (s *SheetSrv) SubmitSheetId(ctx *gin.Context) {
	srv, err := s.GetSheetClient(ctx)
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
	timeSheet := models.Timesheet{}
	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
	} else {
		LatestEntry := models.SheetLastInfo{}
		for _, row := range resp.Values {
			if len(row) > 0 {
				day, err := time.Parse("2/1/2006", row[0].(string))
				if err != nil {
					continue
				}
				if day.After(time.Now()) {
					break
				}

				if weekday := day.Weekday(); weekday == time.Saturday || weekday == time.Sunday {
					continue
				} else if len(row) < 5 {
					break
				} else if row[1] != nil && len(row[1].(string)) > 0 {
					LatestEntry.Date = fmt.Sprintf("%s", row[0])
					LatestEntry.Task = fmt.Sprintf("%s", row[2])
					LatestEntry.Hours = fmt.Sprintf("%s", row[3])
					LatestEntry.Leave = fmt.Sprintf("%s", row[5])
				} else {
					break
				}
			}
			timeSheet.Lastupdate = &LatestEntry
		}
	}
	timeSheet.NewEntry = &models.SheetNewinfo{
		Hours: "8",
		Leave: "No",
	}
	lastDate, err := time.Parse("2/1/2006", timeSheet.Lastupdate.Date)
	if err != nil {
		log.Errorf("some error while pasring the last date %s", err)
	}
	weekday := lastDate.Weekday()
	switch weekday {
	case time.Friday:
		timeSheet.NewEntry.Date = lastDate.AddDate(0, 0, 3).String()
	case time.Saturday:
		timeSheet.NewEntry.Date = lastDate.AddDate(0, 0, 2).String()
	default:
		timeSheet.NewEntry.Date = lastDate.AddDate(0, 0, 1).String()
	}
	ctx.JSON(http.StatusOK, timeSheet)
}

func (s *SheetSrv) GetSheetClient(ctx *gin.Context) (*sheets.Service, error) {
	srv := &sheets.Service{}
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

	srv, err = sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}
	return srv, nil
}
