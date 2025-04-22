package securities

import (
	"context"
	"fmt"
	"net/http"
	"simple-invest/internal/database"
	"simple-invest/internal/servicelog"
	"time"

	"github.com/WLM1ke/gomoex"
)

type bondIndicator struct {
	Isin            string    `json:"isin"`              // ценная бумага
	Facevalue       float64   `json:"facevalue"`         // текущая номинальная стоимость
	Aci             float64   `json:"aci"`               // нкд
	Value           float64   `json:"value"`             // сумма купона
	PercentPrice    float64   `json:"percent_price"`     // цена в процентах
	Price           float64   `json:"price"`             // цена
	DaysToEvent     int64     `json:"days_to_event"`     // дней до события
	RedemtionDate   time.Time `json:"redemtion_date"`    // дата погашения
	SimpleYield     float64   `json:"simple_yield"`      // простая доходоность
	NetSimpleYield  float64   `json:"net_simple_yield"`  // итоговая простая доходность
	CurrentYield    float64   `json:"current_yield"`     // текущая доходность
	NetCurrentYield float64   `json:"net_current_yield"` // итоговая текущая доходность
	TaxRedemption   float64   `json:"tax_redemption"`    // налог при погашении
}

var (
	cl *gomoex.ISSClient
)

const (
	defaultFrom = "1997-01-01"
	defaultTill = "2100-12-31"
)

func init() {
	cl = gomoex.NewISSClient(http.DefaultClient)
}

type Coupon struct {
	Isin             string  `json:"isin"`             // ISIN код
	Name             string  `json:"name"`             // Наименование облигации
	Issuevalue       float64 `json:"issuevalue"`       // Размер выпуска
	Coupondate       string  `json:"coupondate"`       // Дата начала купонного периода
	Recorddate       string  `json:"recorddate"`       // Дата фиксации списка держателей
	Startdate        string  `json:"startdate"`        // Дата начала купонного периода
	Initialfacevalue float64 `json:"initialfacevalue"` // Первоначальная номинальная стоимость
	Facevalue        float64 `json:"facevalue"`        // Номинальная стоимость
	Faceunit         string  `json:"faceunit"`         // Процентная ставка купона
	Value            float64 `json:"value"`            // Сумма купона, в валюте номинала
	Valueprc         float64 `json:"valueprc"`         // Ставка купона, %
	ValueRub         float64 `json:"value_rub"`        // Сумма купона, руб
	Secid            string  `json:"secid"`            // Идентификатор облигации
	PrimaryBoardid   string  `json:"primary_boardid"`  // Идентификатор режима торгов
}

func boardSecuritiesMOEX(engine, market string) ([]gomoex.Security, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var board string
	if market == gomoex.MarketBonds {
		board = "TQCB" // Т+: Облигации - безадрес.
	} else {
		board = gomoex.BoardTQBR // по умолчанию Т+: Акции и ДР — безадресные сделки
	}

	var err error
	table, err := cl.BoardSecurities(ctx, engine, market, board)
	if err != nil {
		servicelog.ErrorLog().Print(err)
		return table, err
	}

	return table, nil

}

func DownloadShares() (err error) {

	secs, err := boardSecuritiesMOEX(gomoex.EngineStock, gomoex.MarketShares)
	if err != nil {
		return err
	}

	existing := make(map[string]bool)
	rows, err := database.DB().Query("SELECT isin FROM securities")
	if err != nil {
		return err
	}
	for rows.Next() {
		var isin string
		err = rows.Scan(&isin)
		if err != nil {
			return err
		}
		existing[isin] = false
	}

	servicelog.InfoLog().Print("Загрузка данных с Мосбиржи: акции")
	var loaded, updated int64
	for _, s := range secs {
		_, ok := existing[s.ISIN]
		if ok {
			// тут обновление
			_, err := database.DB().Exec(`
			UPDATE securities
			SET ticker = $2,
				lotsize = $3,
				board = $4,
				sectype = $5,
				instrument = $6
			WHERE isin = $1;`, s.ISIN, s.Ticker, s.LotSize, s.Board, s.Type, s.Instrument)
			if err != nil {
				return err
			}
			updated++
		} else {
			_, err := database.DB().Exec(`
			INSERT INTO securities (isin, ticker, lotsize, board, sectype, instrument)
			VALUES ($1, $2, $3, $4, $5, $6)`, s.ISIN, s.Ticker, s.LotSize, s.Board, s.Type, s.Instrument)
			if err != nil {
				return err
			}
			fmt.Printf("added security: %s (%s)\n", s.Instrument, s.Ticker)
			loaded++
		}
	}

	servicelog.InfoLog().Printf("Результат закгрузки данных\nзагружено: %d, обновлено: %d", loaded, updated)
	return nil

}

func DownloadBonds() (err error) {

	secs, err := boardSecuritiesMOEX(gomoex.EngineStock, gomoex.MarketBonds)
	if err != nil {
		return err
	}

	existing := make(map[string]bool)
	rows, err := database.DB().Query("SELECT isin FROM securities")
	if err != nil {
		return err
	}
	for rows.Next() {
		var isin string
		err = rows.Scan(&isin)
		if err != nil {
			return err
		}
		existing[isin] = false
	}

	servicelog.InfoLog().Print("Загрузка данных с Мосбиржи: облигации")
	var loaded, updated int64
	for _, s := range secs {
		_, ok := existing[s.ISIN]
		if ok {
			// тут обновление
			updated++
		} else {
			_, err := database.DB().Exec(`
			INSERT INTO securities (isin, ticker, lotsize, board, sectype, instrument)
			VALUES ($1, $2, $3, $4, $5, $6)`, s.ISIN, s.Ticker, s.LotSize, s.Board, s.Type, s.Instrument)
			if err != nil {
				return err
			}
			fmt.Printf("added security: %s (%s)\n", s.Instrument, s.Ticker)
			loaded++
		}
	}

	servicelog.InfoLog().Printf("Результат закгрузки данных\nзагружено: %d, обновлено: %d", loaded, updated)
	return nil

}

func MarketHistory(engine, market, isin, from, till string) ([]gomoex.Quote, error) {

	if from == "" {
		from = defaultFrom
	}

	if till == "" {
		till = defaultTill
	}

	ctx := context.Background()
	quote, err := cl.MarketHistory(ctx, engine, market, isin, from, till)

	return quote, err

}

func Dividends(ctx context.Context, isin string) ([]gomoex.Dividend, error) {
	dividends, err := cl.Dividends(ctx, isin)
	if err != nil {
		return nil, err
	}
	return dividends, nil
}
