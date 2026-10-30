package db

import (
	"context"
	"net/url"
	"time"

	"github.com/Pastir/footballdata/config"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx"
	pgx2 "github.com/jackc/pgx/v4"
)

var (
	Prov ProviderSettings
)

type (
	ProviderSettings struct {
		Id                      int
		ProviderName            string
		GetParams               map[string]interface{}
		Params                  url.Values
		CountryProviderNameData map[string]ProviderCountry
		Endpoints               map[string]DataEndpoints
	}

	DataEndpoints struct {
		Endpoint        string
		Name            string
		TaskName        string
		Additional      map[string]interface{}
		UpdateFrequency time.Duration
		Cycle           string
	}

	ProviderCountry struct {
		Id   int
		Uuid string
		Name string
	}
)

func (pvs *ProviderSettings) Init() error {
	pvs.Endpoints = make(map[string]DataEndpoints)
	pvs.GetParams = make(map[string]interface{})
	pvs.CountryProviderNameData = make(map[string]ProviderCountry)
	pvs.Params = url.Values{}

	err := pvs.GetSettingsProvider()
	if err != nil {
		return err
	}

	err = pvs.GetEndpointsProvider(pvs.Id)
	if err != nil {
		return err
	}

	err = pvs.GetDataFromNameCountryProvider()
	if err != nil {
		return err
	}
	return nil
}

// GetSettingsProvider
// Настройки выбранного провайдера
func (pvs *ProviderSettings) GetSettingsProvider() error {
	// Получаем данные

	query := `SELECT id ,name ,settings->'get_params' AS get_params
              FROM provider
	          WHERE status = 't' AND id = $1`

	rows, err := DBx.PGxPool.Query(context.Background(), query, config.Config.Args.ProviderId)
	if err != nil {
		PGxConnPanic("Query InitData", "GetSettingsProvider", err.(*pgconn.PgError))
		return err.(*pgconn.PgError)
	}

	for rows.Next() {
		err = rows.Scan(&pvs.Id, &pvs.ProviderName, &pvs.GetParams)
		if err != nil {
			PGxScanPanic("Scan InitData", "GetSettingsProvider", err.(pgx2.ScanArgError))
			return err.(pgx2.ScanArgError)
		}
	}

	// GET параметры
	for key, val := range Prov.GetParams {
		Prov.Params.Add(key, val.(string))
	}

	return nil
}

// GetEndpointsProvider
// Установка всех endpoints провайдера
func (pvs *ProviderSettings) GetEndpointsProvider(providerId int) error {

	// Получаем данные
	query := `SELECT endpoint, name, task_name, additional, update_frequency, CASE WHEN cycle IS NULL THEN '' ELSE cycle END  
              FROM provider_endpoints 
              WHERE provider_id = $1`

	rows, err := DBx.PGxPool.Query(context.Background(), query, providerId)
	if err != nil {
		PGxConnPanic("Query InitData", "SetEndpointsProvider", err.(*pgconn.PgError))
		return err.(*pgconn.PgError)
	}

	for rows.Next() {
		var t DataEndpoints
		t.Additional = make(map[string]interface{})

		err = rows.Scan(&t.Endpoint, &t.Name, &t.TaskName, &t.Additional, &t.UpdateFrequency, &t.Cycle)
		if err != nil {
			PGxScanPanic("Scan InitData", "SetEndpointsProvider", err.(pgx2.ScanArgError))
			return err.(pgx2.ScanArgError)
		}

		Prov.Endpoints[t.TaskName] = t
	}

	return nil
}

func (pvs *ProviderSettings) Close() error { return nil }
func (pvs *ProviderSettings) Check() error { return nil }

func (pvs *ProviderSettings) Name() string {
	return "settings provider"
}

// GetDataFromNameCountryProvider
// Возвращает мапу данных стран по имени
func (pvs *ProviderSettings) GetDataFromNameCountryProvider() error {

	query := `SELECT id, name, uid FROM provider_countries WHERE uid NOTNULL`
	rows, err := DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		PGxPanic("InitData", "GetDataFromNameCountryProvider", err.(*pgx.PgError))
		return err.(*pgx.PgError)
	}

	for rows.Next() {
		var pc ProviderCountry

		err = rows.Scan(&pc.Id, &pc.Name, &pc.Uuid)
		if err != nil {
			PGxScanPanic("InitData", "GetDataFromNameCountryProvider", err.(pgx2.ScanArgError))
			return err.(pgx2.ScanArgError)
		}

		pvs.CountryProviderNameData[pc.Name] = pc
	}

	return nil
}
