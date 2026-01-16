// Пакет securities реализует функции по работе с ценными бумагами.
//
// Возможности включают в себя получение данных от Мосбиржи с помощью её API, сохранение полученных данных в БД,
// предоставление финансовых показателей по облигациям, данных по купонам и дивидендам акций.
package securities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"simple-invest/internal/repository"
	"sort"
	"time"

	"github.com/WLM1ke/gomoex"
)

const (
	taxRate   = 0.13 // Ставка налога
	precision = 4    // Точность предоставляемых показателей
)

var (
	cl            *gomoex.ISSClient
	errNoMoexData = errors.New("no moex data provided")
)

type SecuritiesService struct {
	repo repository.Repository
}

func New(repo repository.Repository) *SecuritiesService {
	return &SecuritiesService{repo: repo}
}

// Показатели торгуемой облигации
type bondIndicators struct {
	Isin            string  `json:"isin"`              // Ценная бумага
	FaceValue       float64 `json:"facevalue"`         // Текущая номинальная стоимость
	AccruedInt      float64 `json:"accruedint"`        // НКД
	Coupon          float64 `json:"coupon"`            // Сумма купона
	PercentPrice    float64 `json:"percent_price"`     // Цена в процентах
	Price           float64 `json:"price"`             // Цена
	DaysToEvent     int64   `json:"days_to_event"`     // Дней до события
	MatDate         string  `json:"matdate"`           // Дата погашения
	OfferDate       string  `json:"offerdate"`         // Дата оферты
	SimpleYield     float64 `json:"simple_yield"`      // Простая доходоность
	NetSimpleYield  float64 `json:"net_simple_yield"`  // Итоговая простая доходность
	CurrentYield    float64 `json:"current_yield"`     // Текущая доходность
	NetCurrentYield float64 `json:"net_current_yield"` // Итоговая текущая доходность
	MaturityTax     float64 `json:"maturity_tax"`      // Налог при погашении
}

// Структура выплат облигации
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

// Параметры конкретного купона
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

// Параметры конкретной амортизационной выплаты
type Amortization struct {
	Isin             string    `json:"isin"`             // ISIN код
	Amortdate        string    `json:"amortdate"`        // Дата амортизации
	Facevalue        float64   `json:"facevalue"`        // Номинальная стоимость
	Initialfacevalue float64   `json:"initialfacevalue"` // Первоначальная номинальная стоимость
	Faceunit         string    `json:"faceunit"`         // Валюта
	Value            float64   `json:"value"`            // Сумма амортизации, в валюте номинала
	ValueRub         float64   `json:"value_rub"`        // Сумма амортизации, руб
	Date             time.Time // Дата амортизации в формате time.Time
}

// Структура основных свойств облигации
type Bond struct {
	Isin          string  `json:"isin"`                // ISIN код
	ShortName     string  `json:"shortname"`           // Краткое наименование
	AccruedInt    float64 `json:"accruedint"`          // НКД на дату расчетов, в валюте расчетов
	FaceValue     float64 `json:"facevalue"`           // Номинальная (остаточная) стоимость
	MatDate       string  `json:"matdate"`             // Дата погашения
	CouponPeriod  int32   `json:"couponperiod"`        // Купонный период
	CouponPercent float64 `json:"couponpercent"`       // Ставка купона (уточнить по обл. с переменным купоном и флоатерам)
	CouponValue   float64 `json:"couponvalue"`         // Сумма купона
	SecName       string  `json:"secname"`             // Полное наимнование
	FaceUnit      string  `json:"faceunit"`            // Уточнить (возоможно, валюта)
	OfferDate     string  `json:"offerdate,omitempty"` // Дата оферты
	SettleDate    string  `json:"settledate"`          // Дата расчётов сделки
}

// BondMarketData представляет торговые данные облигации:
type BondMarketData struct {
	Last float64 `json:"last"` //последняя цена сделки
}

func init() {
	cl = gomoex.NewISSClient(http.DefaultClient)
}

// Shares возвращает список акций в виде JSON
func (s *SecuritiesService) Shares() ([]gomoex.Security, error) {
	secs, err := s.repo.GetShares()
	if err != nil {
		return nil, err
	}

	return secs, err
}

// Bonds возвращает список облигаций в виде JSON
func (s *SecuritiesService) Bonds() ([]gomoex.Security, error) {
	secs, err := s.repo.GetBonds()
	if err != nil {
		return nil, err
	}

	return secs, err
}

// DownloadShares получает данные по акциям от Мосбиржи и сохраняет в БД.
func (s *SecuritiesService) DownloadShares() (err error) {
	secs, err := boardSecuritiesMOEX(gomoex.EngineStock, gomoex.MarketShares)
	if err != nil {
		return err
	}

	updated, err := s.repo.UpdateShares(secs)
	if err != nil {
		return err
	}

	log.Printf("Updated: %d", updated)
	return nil
}

// DownloadShares получает данные по облигациям от Мосбиржи и сохраняет в БД.
func (s *SecuritiesService) DownloadBonds() (err error) {

	secs, err := boardSecuritiesMOEX(gomoex.EngineStock, gomoex.MarketBonds)
	if err != nil {
		return err
	}

	updated, err := s.repo.UpdateBonds(secs)
	if err != nil {
		return err
	}

	log.Printf("Updated: %d", updated)
	return nil

}

// Dividends получает данные о дивидидендах акции от Мосбиржи
func (s *SecuritiesService) Dividends(ctx context.Context, isin string) ([]gomoex.Dividend, error) {
	dividends, err := cl.Dividends(ctx, isin)
	if err != nil {
		return nil, err
	}
	return dividends, nil
}

// Coupons получает данные о купонах по облигации от Мосбиржи
func (s *SecuritiesService) Coupons(isin string) ([]Coupon, error) {
	// Получение общего объёма данных и объёма, получаемого за одно обращение
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()
	url := fmt.Sprintf("https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization/%s.json?iss.only=coupons.cursor&iss.meta=off", isin)

	// Выполняем GET-запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Print("Failed read response:", err)
		return nil, err
	}

	type CouponsCursor struct {
		Cursor struct {
			Data [][]int64 `json:"data"`
		} `json:"coupons.cursor"`
	}

	// Парсим JSON
	var couponsCursor CouponsCursor
	err = json.Unmarshal(body, &couponsCursor)
	if err != nil {
		return nil, err
	}

	if len(couponsCursor.Cursor.Data) < 1 || len(couponsCursor.Cursor.Data[0]) < 3 {
		return nil, errors.New("incorrect data structure")
	}

	i := couponsCursor.Cursor.Data[0][0]
	total := couponsCursor.Cursor.Data[0][1]
	pagesize := couponsCursor.Cursor.Data[0][2]

	var coupons []Coupon
	// Последовательное получение блоков данных
	for ; i < total; i += pagesize {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
		defer cancel()
		url := fmt.Sprintf("https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization/%s.json?iss.only=coupons&iss.meta=off&start=%d", isin, i)

		// Выполняем GET-запрос
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err != nil {
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		// Читаем тело ответа
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Failed read response:", err)
			return nil, err
		}

		// Парсим JSON
		var bondPayments BondPayments
		err = json.Unmarshal(body, &bondPayments)
		if err != nil {
			fmt.Println("Failed parse JSON:", err)
			return nil, err
		}

		// Преобразуем данные в структуру Coupon
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
	}

	return coupons, nil
}

// Amortizations получает данные об амортизационных выплатах по облигации от Мосбиржи
func (s *SecuritiesService) Amortizations(isin string) ([]Amortization, error) {
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
		return nil, err
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed read response:", err)
		return nil, err
	}

	// Парсим JSON
	var bondPayments BondPayments
	err = json.Unmarshal(body, &bondPayments)
	if err != nil {
		fmt.Println("Failed parse JSON:", err)
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
			ValueRub:         row[9].(float64),
		}
		amortizations = append(amortizations, amortization)
	}

	return amortizations, nil
}

// BondIndicators возвращает JSON с основными показателями торгуемой облигации
func (s *SecuritiesService) BondIndicators(isin string) (bondIndicators, error) {
	bI := bondIndicators{Isin: isin}

	bond, err := moexBond(isin)
	if err != nil {
		return bI, err
	}

	marketData, err := moexBondMarketData(isin)
	if err != nil {
		return bI, err
	}

	eventDateStr := bond.MatDate
	if bond.OfferDate != "" {
		eventDateStr = bond.OfferDate
	}
	eventDate, err := time.Parse(time.DateOnly, eventDateStr)
	if err != nil {
		return bI, err
	}

	today := time.Now().Truncate(time.Hour * 24)

	var settleDate time.Time
	if bond.SettleDate == "" {
		settleDate = today.AddDate(0, 0, 1)
	} else {
		settleDate, err = time.Parse(time.DateOnly, bond.SettleDate)
		if err != nil {
			return bI, err
		}
	}

	bI.FaceValue = bond.FaceValue
	bI.AccruedInt = bond.AccruedInt
	bI.Coupon = bond.CouponValue
	bI.PercentPrice = marketData.Last
	bI.Price = roundFloat(bond.FaceValue*marketData.Last/100, 2) + bond.AccruedInt
	bI.DaysToEvent = int64(eventDate.Sub(today).Hours() / 24)
	bI.MatDate = bond.MatDate
	bI.OfferDate = bond.OfferDate

	if marketData.Last != 0 {
		bI.CurrentYield = roundFloat(bond.CouponPercent/marketData.Last, precision)
		bI.NetCurrentYield = roundFloat(bond.CouponPercent*bond.FaceValue*(1-taxRate)/bI.Price/100, precision)
	}

	couponsAmount := 0.0
	coupons, err := s.Coupons(isin)
	if err != nil {
		return bI, err
	}
	for _, c := range coupons {
		paymentDate, err := time.Parse(time.DateOnly, c.Coupondate)
		if err != nil {
			return bI, err
		}
		if paymentDate.After(settleDate) {
			couponsAmount += c.Value
		}
	}

	// Для амортизируемых ооблигаций необходимо приведение периода
	netDaysToEvent := float64(bI.DaysToEvent)
	amortizations, err := s.Amortizations(isin)
	if err != nil {
		return bI, err
	}
	if len(amortizations) > 0 {
		netDaysToEvent, err = amortizationsNetPeriod(amortizations, settleDate)
		if err != nil {
			return bI, err
		}
	}

	bI.SimpleYield = roundFloat((couponsAmount+bond.FaceValue-bI.Price)/bI.Price*365/netDaysToEvent, precision)
	creditDate := settleDate.AddDate(3, 0, 0) // дата для ЛДВ
	matTax := 0.0
	if !eventDate.After(creditDate) && bI.Price < bond.FaceValue {
		matTax = roundFloat((bond.FaceValue-bI.Price)*taxRate, 2)
	}
	bI.NetSimpleYield = roundFloat((couponsAmount*(1-taxRate)+bond.FaceValue-matTax-bI.Price)/bI.Price*365/netDaysToEvent, precision)

	bI.MaturityTax = matTax

	return bI, nil
}

func boardSecuritiesMOEX(engine, market string) ([]gomoex.Security, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var board string
	if market == gomoex.MarketBonds {
		board = "TQCB"
	} else {
		board = gomoex.BoardTQBR // по умолчанию Т+: Акции и ДР — безадресные сделки
	}

	var err error
	table, err := cl.BoardSecurities(ctx, engine, market, board)
	if err != nil {
		return table, err
	}

	return table, nil
}

func moexBond(isin string) (Bond, error) {
	b := Bond{Isin: isin}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()
	secProperties := "ISIN,SHORTNAME,ACCRUEDINT,FACEVALUE,MATDATE,COUPONPERIOD,COUPONPERCENT,SECNAME,FACEUNIT,COUPONPERCENT,OFFERDATE,SETTLEDATE,COUPONVALUE"
	url := fmt.Sprintf("https://iss.moex.com/iss/engines/stock/markets/bonds/securities/%s.json?iss.meta=off&iss.only=securities,&securities.columns=%s", isin, secProperties)

	// Выполняем GET-запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return b, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return b, err
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка при чтении ответа:", err)
		return b, err
	}

	// Парсим JSON
	type moexBond struct {
		Securities struct {
			Data [][]interface{}
		} `json:"securities"`
	}

	var mBond moexBond
	err = json.Unmarshal(body, &mBond)
	if err != nil {
		return b, err
	}

	for _, row := range mBond.Securities.Data {
		b.ShortName, _ = row[1].(string)
		b.AccruedInt, _ = row[2].(float64)
		b.FaceValue, _ = row[3].(float64)
		b.MatDate, _ = row[4].(string)
		b.CouponPeriod = int32(row[5].(float64))
		b.CouponPercent, _ = row[6].(float64)
		b.SecName, _ = row[7].(string)
		b.FaceUnit, _ = row[8].(string)
		b.CouponPercent, _ = row[9].(float64)
		b.OfferDate, _ = row[10].(string)
		b.SettleDate, _ = row[11].(string)
		b.CouponValue, _ = row[12].(float64)
	}
	return b, nil
}

func moexBondMarketData(isin string) (BondMarketData, error) {
	var marketData BondMarketData
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()
	secProperties := "LAST"
	url := fmt.Sprintf("https://iss.moex.com/iss/engines/stock/markets/bonds/securities/%s.json?iss.meta=off&iss.only=marketdata&marketdata.columns=%s", isin, secProperties)

	// Выполняем GET-запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return marketData, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return marketData, err
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка при чтении ответа:", err)
		return marketData, err
	}

	type moexMarketData struct {
		MarketData struct {
			Data [][]interface{}
		} `json:"marketdata"`
	}

	var moexData moexMarketData
	err = json.Unmarshal(body, &moexData)
	if err != nil {
		return marketData, err
	}

	for _, row := range moexData.MarketData.Data {
		if row[0] == nil {
			return marketData, errNoMoexData
		}
		var ok bool
		marketData.Last, ok = row[0].(float64)
		if !ok {
			return marketData, fmt.Errorf("cannot convert data %v to float64", row[0])
		}
	}
	return marketData, nil
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func amortizationsNetPeriod(am []Amortization, settleDate time.Time) (float64, error) {
	sort.SliceStable(am, func(i, j int) bool {
		return am[i].Amortdate < am[j].Amortdate
	})

	// netPeriod		- приведённый период
	// periodFaceValue	- непогашенная часть номинала на дату амортизации
	// amValue	- размер амортизационного платежа
	var netPeriod, periodFaceValue, amValue float64
	var prevDate time.Time
	init := false
	for i := range am {
		date, err := time.Parse(time.DateOnly, am[i].Amortdate)
		if err != nil {
			return 0, err
		}
		if date.Compare(settleDate) > 0 {
			if !init {
				netPeriod = date.Sub(settleDate).Hours() / 24
				prevDate = date
				periodFaceValue = am[i].Facevalue
				amValue = am[i].ValueRub
				init = true
				continue
			}
			periodFaceValue -= amValue
			netPeriod += date.Sub(prevDate).Hours() / 24 * periodFaceValue / am[i].Facevalue
			amValue = am[i].ValueRub
			prevDate = date
		}
	}
	return netPeriod, nil
}
