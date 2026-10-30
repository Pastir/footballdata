package queries

import (
	"context"
	"github.com/Pastir/footballdata/db"
	"github.com/jackc/pgconn"
)

var ListTrendsText map[string]string

// GetListTrends - Возвращает тексты для трендов
func GetListTrends() {

	ListTrendsText = make(map[string]string)

	query := `SELECT label, word FROM matchcenter_trends_text`

	//Выполняем запрос
	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic("Init ListTrends", "GetListTrends", err.(*pgconn.PgError))
		return
	}

	for rows.Next() {
		var label string
		var word string

		err = rows.Scan(&label, &word)
		if err != nil {
			db.PGxConnPanic("Init ListTrends", "GetListTrends", err.(*pgconn.PgError))
			return
		}
		ListTrendsText[label] = word
	}
}
