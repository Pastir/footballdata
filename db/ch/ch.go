package ch

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Pastir/footballdata/config"
	"strconv"
)

type (
	ClickHouse struct {
		Connect clickhouse.Conn
	}
)

var CHgo ClickHouse

func (ch *ClickHouse) Init() error {
	var err error
	ctx := context.Background()
	ch.Connect, err = clickhouse.Open(&clickhouse.Options{
		Addr: []string{config.Config.CH.DbHost + ":" + strconv.Itoa(config.Config.CH.DbPort)},
		Auth: clickhouse.Auth{
			Database: config.Config.CH.DbName,
			Username: config.Config.CH.DbUser,
			Password: config.Config.CH.DbPass,
		},
		ClientInfo: clickhouse.ClientInfo{
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: "ch-client", Version: "0.1"},
			},
		},
		Debugf: func(format string, v ...interface{}) {
			fmt.Printf(format, v)
		},
	})
	if err != nil {
		return err
	}

	if err := ch.Connect.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return err
	}
	return nil
}

func (ch *ClickHouse) Close() error {
	ch.Connect.Close()
	return nil
}

func (ch *ClickHouse) Check() error {
	ctx := context.Background()
	if err := ch.Connect.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return err
	}
	return nil
}

func (ch *ClickHouse) Name() string {
	return "clickhouse"
}
