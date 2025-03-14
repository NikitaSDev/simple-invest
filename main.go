package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/WLM1ke/gomoex"
	_ "github.com/lib/pq"
)

var (
	cl       *gomoex.ISSClient
	infoLog  *log.Logger
	errorLog *log.Logger
	DB       *sql.DB
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

	connstr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s",
		"postgres",
		"postgres",
		"invest_db",
		"disable")

	var err error
	DB, err = sql.Open("postgres", connstr)
	if err != nil {
		// errorLog.Panic(fmt.Sprintf("ошибка подключения к базе данных: %s", err.Error()))
		errorLog.Panicf("ошибка подключения к базе данных: %s", err.Error())
	}
	defer DB.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", defaultHandle)
	mux.HandleFunc("/dividends", dividends)
	mux.HandleFunc("/securities", securities)

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	port := ":7540"
	infoLog.Print("Запуск сервера")
	if err := http.ListenAndServe(port, mux); err != nil {
		errorLog.Printf("ошибка запуска сервера: %s", err.Error())
	}

}

func securities(w http.ResponseWriter, req *http.Request) {

	update := req.URL.Query().Get("update")
	if update == "yes" {
		secs, err := boardSecuritiesMOEX()
		if err != nil {
			errorLog.Print(err.Error())
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		// Ticker     string
		// LotSize    int
		// ISIN       string
		// Board      string
		// Type       string
		// Instrument string
		for _, s := range secs {
			_, err := DB.Exec(`
			INSERT INTO securities (ticker, lotsize, isin. board, instrument)
			VALUES ($1, $2, $3, $4, $5, $6)`, s.Ticker, s.LotSize, s.ISIN, s.Board, s.Instrument)
			if err != nil {
				errorLog.Print(err.Error())
				writeError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			infoLog.Printf("added security: %s (%s)", s.Instrument, s.Ticker)
		}
	}

	rows, err := DB.Query(
		`SELECT
	ticker, lotsize, isin, board, instrument
	FROM securities`)

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
		fmt.Print("reading security", s)
	}
	fmt.Println(secs)
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

func boardSecuritiesMOEX() ([]gomoex.Security, error) {

	engines := []string{
		gomoex.EngineStock,
	}

	table := []gomoex.Security{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	market := gomoex.MarketShares

	for _, eng := range engines {

		fmt.Println("market:", market)
		var err error
		table, err = cl.BoardSecurities(ctx, eng, market, gomoex.BoardTQBR)
		if err != nil {
			errorLog.Print(err)
			// writeError(w, "Не удалось получить данные от Мосбиржи", http.StatusServiceUnavailable)
			return table, err
		}

		for _, line := range table {
			fmt.Println(line)
		}

	}
	return table, nil

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
	// w.Write([]byte("Сервер запущен"))
	w.WriteHeader(http.StatusNotFound)
}
