package handler

import (
	"bytes"
	"context"
	"encoding/json"
	firestore "example/timesheet/fireStore"
	"example/timesheet/models"
	"example/timesheet/redis"
	"example/timesheet/utils"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	mail "github.com/go-gomail/gomail"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SheetSrv struct {
	Cache *redis.Cache
	Db    *firestore.FireStore
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
	timeSheet := models.Timesheet{}
	reload := jsonMap["Reload"]
	if reload == nil || !reload.(bool) {
		sheetLastInfo := models.SheetLastInfo{}
		found, _ := s.Cache.CheckDataInCache(context.Background(), sheetId.(string)+"_latest", &sheetLastInfo)
		if found && sheetLastInfo.Date != "" {
			timeSheet.Lastupdate = &sheetLastInfo
		}
	}
	if timeSheet.Lastupdate == nil {
		resp, err := srv.Spreadsheets.Values.Get(sheetId.(string), sheetName.(string)).Do()
		if err != nil {
			utils.AbortWithError(ctx, err)
			return
		}
		if len(resp.Values) == 0 {
			fmt.Println("No data found.")
		} else {
			LatestEntry := models.SheetLastInfo{}
			for i, row := range resp.Values {
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
						LatestEntry.A1Range = fmt.Sprintf("%s!%v:%v", sheetName, i+1, i+1)
					} else {
						break
					}
				}
				timeSheet.Lastupdate = &LatestEntry
			}
			cacheString, err := json.Marshal(LatestEntry)
			if err != nil {
				log.Errorf("error on cache data marshalling := %s", err)
			}
			s.Cache.SetDataInCache(context.Background(), sheetId.(string)+"_latest", cacheString, models.CacheNoExp)
		}
	}
	timeSheet.NewEntry = GetNextUpdateinfo(*timeSheet.Lastupdate, sheetName.(string))
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

func (s *SheetSrv) AddSheetEntry(ctx *gin.Context) {
	sheetEntry := &models.SheetNewinfo{}
	if err := ctx.BindJSON(sheetEntry); err != nil {
		log.Errorf("error on unmarshallig post body :-%s", err)
		utils.AbortWithError(ctx, err)
		return
	}
	srv, err := s.GetSheetClient(ctx)
	if err != nil {
		log.Errorf("error on unmarshallig post body :-%s", err)
		utils.AbortWithError(ctx, err)
		return
	}
	date, err := time.Parse("2006-1-2", sheetEntry.Date)
	if err != nil {
		log.Errorf("error on unmarshallig post body :-%s", err)
		utils.AbortWithError(ctx, err)
		return
	}
	dateEntry := fmt.Sprintf("%v-%v-%v", date.Day(), date.Month(), date.Year())
	_, err = srv.Spreadsheets.Values.Update(sheetEntry.SheetId, sheetEntry.A1Range, &sheets.ValueRange{
		Values: [][]interface{}{
			{dateEntry, date.Weekday().String(), sheetEntry.Task, sheetEntry.Hours, sheetEntry.Leave, sheetEntry.Leave},
		},
	}).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		utils.AbortWithError(ctx, err)
		return
	}
	LatestEntry := models.SheetLastInfo{
		Date:    date.Format("2/1/2006"),
		Task:    sheetEntry.Task,
		Hours:   sheetEntry.Hours,
		Leave:   sheetEntry.Leave,
		A1Range: sheetEntry.A1Range,
	}
	cacheString, err := json.Marshal(LatestEntry)
	if err != nil {
		log.Errorf("error on cache data marshalling := %s", err)
	}
	s.Cache.SetDataInCache(context.Background(), sheetEntry.SheetId+"_latest", cacheString, models.CacheNoExp)
	timeSheet := models.Timesheet{Lastupdate: &LatestEntry}
	timeSheet.NewEntry = GetNextUpdateinfo(LatestEntry, sheetEntry.SheetName)
	if date.Weekday() == time.Friday {
		sheetRange := strings.Split(sheetEntry.A1Range, ":")
		currRow, err := strconv.Atoi(sheetRange[1])
		if err != nil {
			log.Errorf("some error on sheet range - %s", err)
		}
		prevRow := currRow - 6
		range_ := strings.Replace(sheetEntry.A1Range, fmt.Sprintf("%v:", currRow), fmt.Sprintf("%v:", prevRow), 1)
		data := getSheetData(srv, sheetEntry.SheetId, range_)
		table := convertToHTMLTable(data)
		MailData := s.GetFromDB("email_templates", "sheetId", sheetEntry.SheetId,
			[]string{"template", "to", "cc", "name", "project", "client"})
		templateBody := strings.ReplaceAll(fmt.Sprintf("%s", MailData["template"]), "<tableBody></tableBody>", table)
		templateBody = strings.ReplaceAll(templateBody, "%empl_name%", MailData["name"].(string))
		templateBody = strings.ReplaceAll(templateBody, "%client_name%", MailData["client"].(string))
		templateBody = strings.ReplaceAll(templateBody, "%proj_name%", MailData["project"].(string))
		startDate := date.AddDate(0, 0, -7)
		billing_period := fmt.Sprintf("%s to %s", startDate.Format("2006-1-2"), date.Format("2006-01-02"))
		templateBody = strings.ReplaceAll(templateBody, "%billing period%", billing_period)
		templateBody = strings.ReplaceAll(templateBody, "%total_hours%", "40")
		toMails := MailData["to"].([]interface{})
		ccMails := MailData["cc"].(map[string]interface{})
		to := make([]string, 0)
		for _, v := range toMails {
			to = append(to, v.(string))
		}
		cc := make(map[string]string, 0)
		for k, _ := range ccMails {
			cc[k] = ccMails[k].(string)
		}

		Sendmail("midhun.m@techversantinfo.com", to, cc, fmt.Sprintf("weekly timesheet for the period %s", billing_period), templateBody)
	}
	ctx.JSON(200, timeSheet)
}

func GetNextUpdateinfo(currentSheet models.SheetLastInfo, sheetName string) *models.SheetNewinfo {
	NewEntry := &models.SheetNewinfo{
		Hours: "8",
		Leave: "No",
	}
	lastDate, err := time.Parse("2/1/2006", currentSheet.Date)
	if err != nil {
		log.Errorf("some error while pasring the last date %s", err)
	}
	weekday := lastDate.Weekday()
	A1Range := strings.Split(currentSheet.A1Range, "!")
	rowValStr := strings.Split(A1Range[len(A1Range)-1], ":")[0]
	rowVal, err := strconv.Atoi(rowValStr)
	if err != nil {
		panic(err)
	}
	switch weekday {
	case time.Friday:
		NewEntry.Date = lastDate.AddDate(0, 0, 3).String()
		NewEntry.A1Range = fmt.Sprintf("%s!%v:%v", sheetName, rowVal+3, rowVal+3)
	case time.Saturday:
		NewEntry.Date = lastDate.AddDate(0, 0, 2).String()
		NewEntry.A1Range = fmt.Sprintf("%s!%v:%v", sheetName, rowVal+2, rowVal+2)
	default:
		NewEntry.Date = lastDate.AddDate(0, 0, 1).String()
		NewEntry.A1Range = fmt.Sprintf("%s!%v:%v", sheetName, rowVal+1, rowVal+1)
	}
	return NewEntry
}

func (s *SheetSrv) SendStatusMail(template, content, from, to, sub, cc string) (err error) {
	s.Db = firestore.InitDb()
	defer s.Db.Client.Close()
	iter := s.Db.Client.Collection("email_templates").Documents(context.Background())
	mailTemplate := ""
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
			return err
		}
		data := doc.Data()
		if val, ok := data["name"]; ok && val == template {
			mailTemplate = val.(string)
		}
	}
	fmt.Println("mailtemplate", mailTemplate)
	Sendmail("afin.ta@techversantinfo.com", []string{"midhunmnair006@gmail.com"}, nil, "testing", "hello")
	return
}

func (s *SheetSrv) GetTimesheetMailDay(sheetId string) (day string) {
	s.Db = firestore.InitDb()
	defer s.Db.Client.Close()
	iter := s.Db.Client.Collection("timeSheetInfo").Documents(context.Background())
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
			return
		}
		data := doc.Data()
		if val, ok := data["sheetId"]; ok && val == sheetId {
			return fmt.Sprintf("%s", data["mailWeekDay"])
		}
	}
	return
}

func Sendmail(from string, to []string, cc map[string]string, sub, body string) {
	msg := mail.NewMessage()
	msg.SetHeader("From", from)
	msg.SetHeader("To", to...)
	for name, addr := range cc {
		msg.SetAddressHeader("Cc", addr, name)
	}
	msg.SetHeader("Subject", sub)
	msg.SetBody("text/html", body)

	d := mail.NewDialer("smtp.gmail.com", 587, "midhun.m@techversantinfo.com", "midhun@login2021")

	// Send the email to Bob, Cora and Dan.
	if err := d.DialAndSend(msg); err != nil {
		panic(err)
	}
}

func getSheetData(srv *sheets.Service, sheetid, range_ string) [][]interface{} {
	val, err := srv.Spreadsheets.Values.Get(sheetid, range_).Do()
	if err != nil {
		panic(err)
	}
	return val.Values
}
func updateSheetdata(srv *sheets.Service, sheetid string, range_ string, val [][]interface{}) {
	_, err := srv.Spreadsheets.Values.Update(sheetid, range_, &sheets.ValueRange{
		Values: val,
	}).ValueInputOption("RAW").Do()
	if err != nil {
		panic(err)
	}
}

func convertToHTMLTable(data [][]interface{}) string {
	var buffer bytes.Buffer

	// buffer.WriteString("<table border='1' cellpadding='10'>")
	for _, row := range data {
		buffer.WriteString("<tr>")
		for _, cell := range row {
			buffer.WriteString(fmt.Sprintf("<td>%v</td>", cell))
		}
		buffer.WriteString("</tr>")
	}
	// buffer.WriteString("</table>")

	return buffer.String()
}

func (s *SheetSrv) GetFromDB(collection string, key string, keyVal string, reqFileds []string) (returnData map[string]interface{}) {
	s.Db = firestore.InitDb()
	defer s.Db.Client.Close()
	iter := s.Db.Client.Collection(collection).Documents(context.Background())
	returnData = make(map[string]interface{})
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
			return nil
		}
		data := doc.Data()
		if val, ok := data[key]; ok && val == keyVal {
			for _, v := range reqFileds {
				returnData[v] = data[v]
			}
			return returnData
		}
	}
	return nil
}
