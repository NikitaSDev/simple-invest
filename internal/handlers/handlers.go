package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"simple-invest/internal/database"
	"simple-invest/internal/securities"
	"simple-invest/internal/servicelog"

	"github.com/WLM1ke/gomoex"
)

func DefaultHandle(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Сервер запущен"))
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

	writeResponse(w, resp)

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

	resp, err := json.Marshal(secs)
	if err != nil {
		servicelog.ErrorLog().Print(err)
		writeError(w, "Не удалось сериализовать данные", http.StatusInternalServerError)
		return
	}

	writeResponse(w, resp)
}

func Coupons(w http.ResponseWriter, req *http.Request) {
	isin := req.URL.Query().Get("isin")
	if isin == "" {
		err := errors.New("не укаазан ISIN")
		servicelog.ErrorLog().Print(err)
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	coupons, err := securities.Coupons(isin)
	if err != nil {
		servicelog.ErrorLog().Print(err)
		writeError(w, "Не удалось получить данные от Мосбиржи", http.StatusServiceUnavailable)
		return
	}

	resp, err := json.Marshal(coupons)
	if err != nil {
		servicelog.ErrorLog().Print(err)
		writeError(w, "Не удалось сериализовать данные", http.StatusInternalServerError)
		return
	}

	writeResponse(w, resp)
}

func Amortizations(w http.ResponseWriter, req *http.Request) {
	isin := req.URL.Query().Get("isin")
	if isin == "" {
		err := errors.New("не укаазан ISIN")
		servicelog.ErrorLog().Print(err)
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	amortizations, err := securities.Amortizations(isin)
	if err != nil {
		servicelog.ErrorLog().Print(err)
		writeError(w, "Не удалось получить данные от Мосбиржи", http.StatusServiceUnavailable)
		return
	}

	resp, err := json.Marshal(amortizations)
	if err != nil {
		servicelog.ErrorLog().Print(err)
		writeError(w, "Не удалось сериализовать данные", http.StatusInternalServerError)
		return
	}

	writeResponse(w, resp)
}

func MarketHistory(w http.ResponseWriter, req *http.Request) {
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

func BondIndicators(w http.ResponseWriter, req *http.Request) {
	isin := req.URL.Query().Get("isin")
	if isin == "" {
		writeError(w, "Не указана ценная бумага", http.StatusBadRequest)
		return
	}

	bondIndicators, err := securities.BondIndicators(isin)
	if err != nil {
		servicelog.ErrorLog().Print(err)
		writeError(w, "Не удалось получить данные от Мосбиржи", http.StatusServiceUnavailable)
		return
	}

	resp, err := json.Marshal(bondIndicators)
	if err != nil {
		servicelog.ErrorLog().Print(err)
		writeError(w, "Не удалось сериализовать данные", http.StatusInternalServerError)
		return
	}

	writeResponse(w, resp)

}

func writeResponse(w http.ResponseWriter, resp []byte) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
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
