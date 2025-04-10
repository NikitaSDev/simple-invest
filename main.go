package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/WLM1ke/gomoex"
	_ "github.com/lib/pq"
)

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

type APIResponse struct {
	Coupons struct {
		Columns []string        `json:"columns"` // Названия колонок
		Data    [][]interface{} `json:"data"`    // Данные
	} `json:"coupons"`
}

var (
	cl       *gomoex.ISSClient
	infoLog  *log.Logger
	errorLog *log.Logger
	DB       *sql.DB
)

const (
	ofz26238 = "SU26238RMFS4"
)

/*
	Торговые системы
	EngineStock    = "stock"    // Фондовый рынок и рынок депозитов
	EngineCurrency = "currency" // Валютный рынок
	EngineFutures  = "futures"  // Срочный рынок

	Рынки
	MarketIndex         = "index"         // Индексы фондового рынка
	MarketShares        = "shares"        // Рынок акций
	MarketBonds         = "bonds"         // Рынок облигаций
	MarketForeignShares = "foreignshares" // Иностранные ценные бумаги
	MarketSelt          = "selt"          // Биржевые сделки с ЦК
	MarketFutures       = "futures"       // Поставочные фьючерсы
	MarketFORTS         = "forts"         // ФОРТС
	MarketOptions       = "options"       // Опционы ФОРТС

	Режимы торгов
	BoardTQBR = "TQBR" // Т+: Акции и ДР — безадресные сделки
	BoardTQTF = "TQTF" // Т+: ETF — безадресные сделки
	BoardFQBR = "FQBR" // Т+ Иностранные Акции и ДР — безадресные сделки
*/

func main() {

	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	defer infoLog.Print("Сервер остановлен")

	cl = gomoex.NewISSClient(http.DefaultClient)

	var err error
	DB, err = connectDB()
	if err != nil {
		errorLog.Panicf("ошибка подключения к базе данных: %s", err.Error())
		return
	}
	defer DB.Close()
	infoLog.Print("Подключение к базе данных установлено")

	mux := http.NewServeMux()
	mux.HandleFunc("/", defaultHandle)
	mux.HandleFunc("/dividends", dividends)
	mux.HandleFunc("/shares", shares)
	mux.HandleFunc("/coupons", coupons)

	fileServer := http.FileServer(http.Dir("./ui/static/")) // Проверить или удалить
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	port := ":7540"
	infoLog.Print("Запуск сервера")
	if err := http.ListenAndServe(port, mux); err != nil {
		errorLog.Printf("ошибка запуска сервера: %s", err.Error())
	}

}

func connectDB() (*sql.DB, error) {

	connstr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s",
		"postgres",
		"postgres",
		"invest_db",
		"disable")
	db, err := sql.Open("postgres", connstr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func shares(w http.ResponseWriter, req *http.Request) {

	update := req.URL.Query().Get("update")
	if update == "yes" {
		if err := downloadShares(); err != nil {
			errorLog.Print(err.Error())
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
	}

	rows, err := DB.Query("SELECT ticker, lotsize, isin, board, instrument FROM securities")

	if err != nil {
		errorLog.Print(err.Error())
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	secs := []gomoex.Security{}
	for rows.Next() {
		s := gomoex.Security{}
		err := rows.Scan(&s.Ticker, &s.LotSize, &s.ISIN, &s.Board, &s.Instrument)
		if err != nil {
			errorLog.Print(err.Error())
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		secs = append(secs, s)
		// fmt.Println("reading security", s)
	}
	// fmt.Println(secs)
}

func dividends(w http.ResponseWriter, req *http.Request) {

	security := req.URL.Query().Get("security")
	if security == "" {
		writeError(w, "Не указана ценная бумага", http.StatusBadRequest)
	}

	divs, err := cl.Dividends(context.Background(), security)
	if err != nil {
		errorLog.Print(err)
		writeError(w, "Не удалось получить данные от Мосбиржи", http.StatusServiceUnavailable)
		return
	}

	resp, err := json.Marshal(divs)
	if err != nil {
		errorLog.Print(err)
		writeError(w, "Не удалось сериализовать данные", http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)

}

func boardSecuritiesMOEX(engine string) ([]gomoex.Security, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	market := gomoex.MarketShares

	var err error
	table, err := cl.BoardSecurities(ctx, engine, market, gomoex.BoardTQBR)
	if err != nil {
		errorLog.Print(err)
		return table, err
	}

	return table, nil

}

func downloadShares() (err error) {

	secs, err := boardSecuritiesMOEX(gomoex.EngineStock)
	if err != nil {
		return err
	}

	existing := make(map[string]bool)
	rows, err := DB.Query("SELECT isin FROM securities")
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

	infoLog.Print("Загрузка данных с Мосбиржи: акции")
	var loaded, updated int64
	for _, s := range secs {
		_, ok := existing[s.ISIN]
		if ok {
			// тут обновление
			updated++
		} else {
			_, err := DB.Exec(`
			INSERT INTO securities (isin, ticker, lotsize, board, sectype, instrument)
			VALUES ($1, $2, $3, $4, $5, $6)`, s.ISIN, s.Ticker, s.LotSize, s.Board, s.Type, s.Instrument)
			if err != nil {
				return err
			}
			fmt.Printf("added security: %s (%s)", s.Instrument, s.Ticker)
			loaded++
		}
	}
	infoLog.Printf("Результат закгрузки данных\nзагружено: %d, обновлено: %d", loaded, updated)
	return nil

}

func downloadBonds() {

}

func coupons(w http.ResponseWriter, req *http.Request) {
	isin := req.URL.Query().Get("isin")
	if isin == "" {
		isin = ofz26238
	}
	url := fmt.Sprintf("https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization/%s.json", isin)

	// Проверить варианты ресурса
	// https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization/SU26209RMFS5.json?from=%7BdateString%7D&iss.only=coupons,amortizations&iss.meta=off
	// https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization?from=2020-02-01&till=2020-02-20&start=0&limit=100&iss.only=amortizations,coupons
	// https://iss.moex.com/iss/securities/RU000A0JXQ85/bondization.json?iss.json=extended&iss.meta=off&iss.only=coupons&lang=ru&limit=unlimited

	// Выполняем GET-запрос
	resp, err := http.Get(url)
	if err != nil {
		// fmt.Println("Ошибка при выполнении запроса:", err)
		errorLog.Print(err.Error())
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка при чтении ответа:", err)
		return
	}

	// Парсим JSON
	var apiResponse APIResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		fmt.Println("Ошибка при парсинге JSON:", err)
		return
	}

	// Преобразуем данные в структуру Coupon
	var coupons []Coupon
	for _, row := range apiResponse.Coupons.Data {
		fmt.Println(row...)
		coupon := Coupon{
			Isin:             row[0].(string),
			Name:             row[1].(string),
			Issuevalue:       row[2].(float64),
			Coupondate:       row[3].(string),
			Recorddate:       row[4].(string),
			Startdate:        row[5].(string),
			Initialfacevalue: row[6].(float64),
			Facevalue:        row[7].(float64),
			Faceunit:         row[8].(string),
			Value:            row[9].(float64),
			Valueprc:         row[10].(float64),
			ValueRub:         row[11].(float64),
			Secid:            row[12].(string),
			PrimaryBoardid:   row[13].(string),
		}
		coupons = append(coupons, coupon)
	}

	// Выводим данные о купонах
	for _, coupon := range coupons {
		fmt.Printf("Облигация: %s\n", coupon.Name)
		fmt.Printf("Дата выплаты: %s\n", coupon.Coupondate)
		fmt.Printf("Размер купона: %.2f %s\n", coupon.Value, coupon.Faceunit)
		fmt.Printf("Процентная ставка: %.2f%%\n", coupon.Valueprc)
		fmt.Println("---")
	}
}

func writeError(w http.ResponseWriter, textErr string, status int) {
	responseErr := map[string]string{
		"error": textErr,
	}
	errJSON, err := json.Marshal(responseErr)
	if err != nil {
		log.Fatal(err)
	}
	w.WriteHeader(status)
	w.Write(errJSON)

}

func defaultHandle(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Сервер запущен"))
	// w.WriteHeader(http.StatusNotFound)
}
