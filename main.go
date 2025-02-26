package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/WLM1ke/gomoex"
)

var (
	cl       *gomoex.ISSClient
	infoLog  *log.Logger
	errorLog *log.Logger
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

	infoLog = log.New(os.Stdout, "INFO", log.Ldate|log.Ltime)
	errorLog = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	cl = gomoex.NewISSClient(http.DefaultClient)

	infoLog.Print("Запуск сервера")

	mux := http.NewServeMux()
	mux.HandleFunc("/dividends", dividends)
	mux.HandleFunc("/boardsecurities", boardSecuritiesMOEX)

}

func dividends(w http.ResponseWriter, req *http.Request) {

	security := req.URL.Query().Get("security")
	if security == "" {
		setError(w, "Не указана ценная бумага", http.StatusBadRequest)
	}

	divs, err := cl.Dividends(context.Background(), security)
	if err != nil {
		errorLog.Print(err)
		setError(w, "Не удалось получить данные от Мосбиржи", http.StatusServiceUnavailable)
		return
	}

	for _, div := range divs {
		fmt.Println(div)
	}

}

func boardSecuritiesMOEX(w http.ResponseWriter, req *http.Request) {

	engines := []string{
		gomoex.EngineStock,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	market := gomoex.MarketShares

	for _, eng := range engines {

		fmt.Println("market:", market)
		table, err := cl.BoardSecurities(ctx, eng, market, gomoex.BoardTQBR)
		if err != nil {
			errorLog.Print(err)
			setError(w, "Не удалось получить данные от Мосбиржи", http.StatusServiceUnavailable)
			return
		}

		for _, line := range table {
			fmt.Println(line)
		}

	}

}

func setError(w http.ResponseWriter, textErr string, status int) {
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
