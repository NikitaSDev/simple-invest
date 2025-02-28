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

	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	defer infoLog.Print("Сервер остановлен")

	cl = gomoex.NewISSClient(http.DefaultClient)

	mux := http.NewServeMux()
	mux.HandleFunc("/", defaultHandle)
	mux.HandleFunc("/dividends", dividends)
	mux.HandleFunc("/boardsecurities", boardSecuritiesMOEX)

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	port := ":7540"
	infoLog.Print("Запуск сервера")
	if err := http.ListenAndServe(port, mux); err != nil {
		errorLog.Printf("ошибка запуска сервера: %s", err.Error())
	}

}

func downloadSecuritiesMOEX() error {

	engine := gomoex.EngineStock

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	market := gomoex.MarketShares

	infoLog.Printf("загрузка данных Мосбиржи: %s", market)
	table, err := cl.BoardSecurities(ctx, engine, market, gomoex.BoardTQBR)
	if err != nil {
		errorLog.Print(err)
		return err
	}

	for _, line := range table {
		fmt.Println(line)
	}
	return nil

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
			writeError(w, "Не удалось получить данные от Мосбиржи", http.StatusServiceUnavailable)
			return
		}

		for _, line := range table {
			fmt.Println(line)
		}

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
}
