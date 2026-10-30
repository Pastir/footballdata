package utility

import (
	"github.com/OneOfOne/xxhash"
	"reflect"
	"strconv"
	"strings"
)

type (
	HashData struct {
		MarkertId int
		Market    *string
		BkId      int
		MatchId   int
		TeamId    int
		Value     *string
		Label     *string
		Handicap  *string
		Total     *string
		Context   *string
		WhatText  *string
		hash      uint64
	}
)

func (h *HashData) Init() {
	h.Value = new(string)
	h.Label = new(string)
	h.Total = new(string)
	h.Handicap = new(string)
	h.Context = new(string)
	h.Market = new(string)
	h.WhatText = new(string)
}

func GenerateHash64(data HashData) uint64 {
	var StrOdds strings.Builder
	StrOdds.WriteString(strconv.Itoa(data.MarkertId))
	StrOdds.WriteString(strconv.Itoa(data.BkId))
	StrOdds.WriteString(strconv.Itoa(data.MatchId))
	StrOdds.WriteString(strconv.Itoa(data.TeamId))
	//Handicap
	if reflect.ValueOf(data.Handicap).IsNil() == false {
		StrOdds.WriteString(*data.Handicap)
	} else {
		StrOdds.WriteString("h")
	}
	//Total
	if reflect.ValueOf(data.Total).IsNil() == false {
		StrOdds.WriteString(*data.Total)
	} else {
		StrOdds.WriteString("t")
	}
	//Label
	StrOdds.WriteString(*data.Label)

	//Context
	if reflect.ValueOf(data.Context).IsNil() == false {
		StrOdds.WriteString(*data.Context)
	} else {
		StrOdds.WriteString("c")
	}

	//WhatText
	if reflect.ValueOf(data.WhatText).IsNil() == false {
		StrOdds.WriteString(*data.WhatText)
	} else {
		StrOdds.WriteString("wt")
	}

	//Market string
	if reflect.ValueOf(data.Market).IsNil() == false {
		StrOdds.WriteString(*data.Market)
	} else {
		StrOdds.WriteString("m")
	}

	hashLineUps := xxhash.New64()
	hashLineUps.WriteString(StrOdds.String())
	return hashLineUps.Sum64()
}
