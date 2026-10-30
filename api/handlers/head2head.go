package handler

import (
	"github.com/Pastir/footballdata/api/statistics"
	"github.com/Pastir/footballdata/config"
)

func UpsertHead2Head(taskData config.TaskData) {
	statistics.Matches(taskData)
}
