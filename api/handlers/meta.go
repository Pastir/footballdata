package handler

import (
	"context"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/jackc/pgconn"
	pgx2 "github.com/jackc/pgx/v4"
	"github.com/valyala/fastjson"
	"strings"
	"time"
)

type (
	PaginationData struct {
		CurrentPage *int
		HasMore     *bool
	}

	RateLimit struct {
		ResetsInSeconds *int64
		Remaining       *int
	}
)

func (pd *PaginationData) Init() {
	pd.CurrentPage = new(int)
	pd.HasMore = new(bool)
}

func (rl *RateLimit) Init() {
	rl.ResetsInSeconds = new(int64)
	rl.Remaining = new(int)
}

// PaginationHandler Функция обрабатывает ответ провайдера
func PaginationHandler(p *fastjson.Value, pgData *PaginationData) {
	*pgData.HasMore = p.GetBool("has_more")
	*pgData.CurrentPage = p.GetInt("current_page")
}

func RateLimitHandler(p *fastjson.Value, rateLimitData *RateLimit) {
	*rateLimitData.Remaining = p.GetInt("remaining")
	*rateLimitData.ResetsInSeconds = p.GetInt64("resets_in_seconds")
}

// TrimBetweenDateLimitPeriod - splits a long period into parts (allowed by the limit)
func TrimBetweenDateLimitPeriod(argsData config.LineArgs) ([][]string, error) {
	var betweenDateItog [][]string
	var betweenDate []string
	betweenDate = append(betweenDate, argsData.StartDate)
	betweenDate = append(betweenDate, argsData.EndDate)

	startDate, err := time.Parse("2006-01-02", betweenDate[0])
	if err != nil {
		return nil, err
	}
	endDate, _ := time.Parse("2006-01-02", betweenDate[1])
	if err != nil {
		return nil, err
	}

	limitDate := startDate.AddDate(0, 0, 31)
	betweenDateItog = append(betweenDateItog, []string{startDate.Format("2006-01-02"), limitDate.Format("2006-01-02")})
	flag := false

	for {
		startDate = limitDate.AddDate(0, 0, 1)
		limitDate = limitDate.AddDate(0, 0, 31)

		if limitDate.After(endDate) {
			limitDate = endDate
			flag = true
		}

		betweenDateItog = append(betweenDateItog, []string{startDate.Format("2006-01-02"), limitDate.Format("2006-01-02")})

		if flag {
			break
		}
	}
	return betweenDateItog, nil
}

func SetSeasonsId(taskData config.TaskData) ([]int, error) {

	var seasonId []int
	var andWhere []string
	var sectionWhere string

	query := `SELECT ps.id FROM provider_seasons AS ps LEFT JOIN provider_leagues AS pl ON (ps.league_id = pl.id) `

	andWhere = append(andWhere, " pl.active ")

	if strings.Compare(taskData.GetSeasonsId(), "") != 0 {
		andWhere = append(andWhere, " ps.id IN ("+taskData.GetSeasonsId()+") ")
	}

	if taskData.GetCurrentSeason() {
		andWhere = append(andWhere, " is_current ")
	}

	sizeWhere := len(andWhere)

	if sizeWhere > 0 {
		sectionWhere = " WHERE "
	}
	for k, v := range andWhere {
		if k == sizeWhere-1 {
			sectionWhere += v
		} else {
			sectionWhere += v + "AND"
		}
	}

	rows, err := db.DBx.PGxPool.Query(context.Background(), query+sectionWhere)
	if err != nil {
		db.PGxConnPanic(taskData.Name(), "SetSeasonsId", err.(*pgconn.PgError))
	}

	for rows.Next() {
		var t int
		err = rows.Scan(&t)
		if err != nil {
			db.PGxScanPanic(taskData.Name(), "SetSeasonsId", err.(pgx2.ScanArgError))
		}

		seasonId = append(seasonId, t)
	}

	return seasonId, nil
}
