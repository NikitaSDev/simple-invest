package securities

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"simple-invest/internal/database"
	"simple-invest/internal/servicelog"
	"time"

	"github.com/WLM1ke/gomoex"
)

const (
	defaultFrom = "1997-01-01"
	defaultTill = "2100-12-31"
	taxRate     = 0.13
)

var (
	cl *gomoex.ISSClient
)

type bondIndicators struct {
	Isin            string    `json:"isin"`              // Ценная бумага
	Facevalue       float64   `json:"facevalue"`         // Текущая номинальная стоимость
	Aci             float64   `json:"aci"`               // НКД
	Value           float64   `json:"value"`             // Сумма купона
	PercentPrice    float64   `json:"percent_price"`     // Цена в процентах
	Price           float64   `json:"price"`             // Цена
	DaysToEvent     int64     `json:"days_to_event"`     // Дней до события
	RedemtionDate   time.Time `json:"redemtion_date"`    // Дата погашения
	SimpleYield     float64   `json:"simple_yield"`      // Простая доходоность
	NetSimpleYield  float64   `json:"net_simple_yield"`  // Итоговая простая доходность
	CurrentYield    float64   `json:"current_yield"`     // Текущая доходность
	NetCurrentYield float64   `json:"net_current_yield"` // Итоговая текущая доходность
	TaxRedemption   float64   `json:"tax_redemption"`    // Налог при погашении
}

type BondPayments struct {
	Coupons struct {
		Columns []string        `json:"columns"` // Названия колонок
		Data    [][]interface{} `json:"data"`    // Данные
	} `json:"coupons"`
	Amortizations struct {
		Columns []string        `json:"columns"` // Названия колонок
		Data    [][]interface{} `json:"data"`
	} `json:"amortizations"`
}

type Coupon struct {
	Isin             string  `json:"isin"`             // ISIN код
	Coupondate       string  `json:"coupondate"`       // Дата выплаты купона
	Recorddate       string  `json:"recorddate"`       // Дата фиксации списка держателей
	Initialfacevalue float64 `json:"initialfacevalue"` // Первоначальная номинальная стоимость
	Facevalue        float64 `json:"facevalue"`        // Номинальная стоимость
	Faceunit         string  `json:"faceunit"`         // Валюта
	Value            float64 `json:"value"`            // Сумма купона, в валюте номинала
	Valueprc         float64 `json:"valueprc"`         // Ставка купона, %
	ValueRub         float64 `json:"value_rub"`        // Сумма купона, руб
}

type Amortization struct {
	Isin             string  `json:"isin"`             // ISIN код
	Amortdate        string  `json:"amortdate"`        // Дата амортизации
	Facevalue        float64 `json:"facevalue"`        // Номинальная стоимость
	Initialfacevalue float64 `json:"initialfacevalue"` // Первоначальная номинальная стоимость
	Faceunit         string  `json:"faceunit"`         // Валюта
	Value            float64 `json:"value"`            // Сумма амортизации, в валюте номинала
	Value_rub        float64 `json:"value_rub"`        // Сумма амортизации, руб
}

type Bond struct {
	Isin          string  `json:"isin"`          // ISIN код
	ShortName     string  `json:"SHORTNAME"`     // Краткое наименование
	Accruedint    string  `json:"ACCRUEDINT"`    // НКД на дату расчетов, в валюте расчетов
	FaceValue     float64 `json:"FACEVALUE"`     // Номинальная (остаточная) стоимость
	MatDate       string  `json:"MATDATE"`       // Дата погашения
	CouponPeriod  int16   `json:"COUPONPERIOD"`  // Купонный период
	SecName       string  `json:"SECNAME"`       // Полное наимнование
	FaceUnit      string  `json:"FACEUNIT"`      // Уточнить (возоможно, валюта)
	CouponPercent float64 `json:"COUPONPERCENT"` // Ставка купона (уточнить по обл. с переменным купоном и флоатерам)
	OfferDate     string  `json:"OFFERDATE"`     // Дата оферты
	SettleDate    string  `json:"SETTLEDATE"`    // Дата расчётов сделки
}

func init() {
	cl = gomoex.NewISSClient(http.DefaultClient)
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

func Coupons(isin string) ([]Coupon, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()
	url := fmt.Sprintf("https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization/%s.json?iss.only=coupons", isin)

	// Выполняем GET-запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		servicelog.ErrorLog().Print(err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка при чтении ответа:", err)
		return nil, err
	}

	// Парсим JSON
	var bondPayments BondPayments
	err = json.Unmarshal(body, &bondPayments)
	if err != nil {
		fmt.Println("Ошибка парсинга JSON:", err)
		return nil, err
	}

	// Преобразуем данные в структуру Coupon
	var coupons []Coupon
	for _, row := range bondPayments.Coupons.Data {
		coupon := Coupon{
			Isin:             row[0].(string),
			Coupondate:       row[3].(string),
			Recorddate:       row[4].(string),
			Initialfacevalue: row[6].(float64),
			Facevalue:        row[7].(float64),
			Faceunit:         row[8].(string),
			Value:            row[9].(float64),
			Valueprc:         row[10].(float64),
			ValueRub:         row[11].(float64),
		}
		coupons = append(coupons, coupon)
	}

	return coupons, nil
}

func Amortizations(isin string) ([]Amortization, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()

	url := fmt.Sprintf("https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization/%s.json?iss.only=amortizations", isin)

	// Выполняем GET-запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		servicelog.ErrorLog().Print(err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка при чтении ответа:", err)
		return nil, err
	}

	// Парсим JSON
	var bondPayments BondPayments
	err = json.Unmarshal(body, &bondPayments)
	if err != nil {
		fmt.Println("Ошибка парсинга JSON:", err)
		return nil, err
	}

	// Преобразуем полученные по API данные в структуру Amortization
	var amortizations []Amortization
	for _, row := range bondPayments.Amortizations.Data {
		amortization := Amortization{
			Isin:             row[0].(string),
			Amortdate:        row[3].(string),
			Facevalue:        row[4].(float64),
			Initialfacevalue: row[5].(float64),
			Faceunit:         row[6].(string),
			Value:            row[8].(float64),
			Value_rub:        row[9].(float64),
		}
		amortizations = append(amortizations, amortization)
	}

	return amortizations, nil
}

func BondIndicators(isin string) (bondIndicators, error) {
	bI := bondIndicators{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()

	url := fmt.Sprintf("https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization/%s.json?iss.only=amortizations", isin)

	// Выполняем GET-запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return bI, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		servicelog.ErrorLog().Print(err.Error())
		return bI, err
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка при чтении ответа:", err)
		return bI, err
	}

	// Парсим JSON

	return bI, nil
}
