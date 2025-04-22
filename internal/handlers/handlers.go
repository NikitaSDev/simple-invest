package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"simple-invest/internal/database"
	"simple-invest/internal/securities"
	"simple-invest/internal/servicelog"

	"github.com/WLM1ke/gomoex"
)

type APIResponse struct {
	Coupons struct {
		Columns []string        `json:"columns"` // Названия колонок
		Data    [][]interface{} `json:"data"`    // Данные
	} `json:"coupons"`
}

func DefaultHandle(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Сервер запущен"))
}

func Dividends(w http.ResponseWriter, req *http.Request) {

	isin := req.URL.Query().Get("isin")
	if isin == "" {
		writeError(w, "Не указана ценная бумага", http.StatusBadRequest)
		return
	}

	divs, err := securities.Dividends(context.Background(), isin)
	if err != nil {
		servicelog.ErrorLog().Print(err)
		writeError(w, "Не удалось получить данные от Мосбиржи", http.StatusServiceUnavailable)
		return
	}

	resp, err := json.Marshal(divs)
	if err != nil {
		servicelog.ErrorLog().Print(err)
		writeError(w, "Не удалось сериализовать данные", http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)

}

func Shares(w http.ResponseWriter, req *http.Request) {

	update := req.URL.Query().Get("update")
	if update == "yes" {
		if err := securities.DownloadShares(); err != nil {
			servicelog.ErrorLog().Print(err.Error())
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
	}

	rows, err := database.DB().Query("SELECT ticker, lotsize, isin, board, instrument FROM securities")

	if err != nil {
		servicelog.ErrorLog().Print(err.Error())
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	secs := []gomoex.Security{}
	for rows.Next() {
		s := gomoex.Security{}
		err := rows.Scan(&s.Ticker, &s.LotSize, &s.ISIN, &s.Board, &s.Instrument)
		if err != nil {
			servicelog.ErrorLog().Print(err.Error())
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		secs = append(secs, s)
	}
}

func Bonds(w http.ResponseWriter, req *http.Request) {
	update := req.URL.Query().Get("update")
	if update == "yes" {
		if err := securities.DownloadBonds(); err != nil {
			servicelog.ErrorLog().Print(err.Error())
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
	}

	rows, err := database.DB().Query("SELECT ticker, lotsize, isin, board, instrument FROM securities")

	if err != nil {
		servicelog.ErrorLog().Print(err.Error())
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	secs := []gomoex.Security{}
	for rows.Next() {
		s := gomoex.Security{}
		err := rows.Scan(&s.Ticker, &s.LotSize, &s.ISIN, &s.Board, &s.Instrument)
		if err != nil {
			servicelog.ErrorLog().Print(err.Error())
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		secs = append(secs, s)
	}
}

func Coupons(w http.ResponseWriter, req *http.Request) {
	isin := req.URL.Query().Get("isin")
	if isin == "" {
		err := errors.New("не укаазан ISIN")
		servicelog.ErrorLog().Print(err)
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	url := fmt.Sprintf("https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization/%s.json", isin)

	// Проверить варианты ресурса
	// https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization/SU26209RMFS5.json?from=%7BdateString%7D&iss.only=coupons,amortizations&iss.meta=off
	// https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization?from=2020-02-01&till=2020-02-20&start=0&limit=100&iss.only=amortizations,coupons
	// https://iss.moex.com/iss/securities/RU000A0JXQ85/bondization.json?iss.json=extended&iss.meta=off&iss.only=coupons&lang=ru&limit=unlimited

	// Выполняем GET-запрос
	resp, err := http.Get(url)
	if err != nil {
		servicelog.ErrorLog().Print(err.Error())
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
	var coupons []securities.Coupon
	for _, row := range apiResponse.Coupons.Data {
		fmt.Println(row...)
		coupon := securities.Coupon{
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

func MarketHistori(w http.ResponseWriter, req *http.Request) {

	isin := req.URL.Query().Get("isin")
	if isin == "" {
		writeError(w, "Не указана ценная бумага", http.StatusBadRequest)
		return
	}

	quote, err := securities.MarketHistory(gomoex.EngineStock, gomoex.MarketBonds, isin, "", "")
	if err != nil {
		servicelog.ErrorLog().Print(err.Error())
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, dayPrice := range quote {
		fmt.Printf("Дата: %v, открытие: %f, закрытие: %f, мин. %f, макс %f\n", dayPrice.Date, dayPrice.Open, dayPrice.Close, dayPrice.Low, dayPrice.High)
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
