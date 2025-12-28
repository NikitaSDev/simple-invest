package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"simple-invest/internal/securities"
	"simple-invest/internal/servicelog"
)

type Handler struct {
	service *securities.SecuritiesService
}

func New(service *securities.SecuritiesService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) DefaultHandle(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Сервер запущен"))
}

func (h *Handler) Shares(w http.ResponseWriter, req *http.Request) {
	update := req.URL.Query().Get("update")
	if update == "yes" {
		if err := h.service.DownloadShares(); err != nil {
			servicelog.ErrorLog().Print(err.Error())
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
	}

	secs, err := h.service.Shares()
	if err != nil {
		servicelog.ErrorLog().Print(err.Error())
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(secs)
	if err != nil {
		servicelog.ErrorLog().Print(err)
		writeError(w, "Не удалось сериализовать данные", http.StatusInternalServerError)
		return
	}

	writeResponse(w, resp)
}

func (h *Handler) Dividends(w http.ResponseWriter, req *http.Request) {
	isin := req.URL.Query().Get("ticker")
	if isin == "" {
		writeError(w, "Не указан код ценной бумаги", http.StatusBadRequest)
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

func (h *Handler) Bonds(w http.ResponseWriter, req *http.Request) {
	update := req.URL.Query().Get("update")
	if update == "yes" {
		if err := h.service.DownloadBonds(); err != nil {
			servicelog.ErrorLog().Print(err.Error())
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
	}

	secs, err := h.service.Bonds()
	if err != nil {
		servicelog.ErrorLog().Print(err.Error())
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(secs)
	if err != nil {
		servicelog.ErrorLog().Print(err)
		writeError(w, "Не удалось сериализовать данные", http.StatusInternalServerError)
		return
	}

	// rows, err := database.DB().Query("SELECT ticker, lotsize, isin, board, instrument FROM securities")

	// if err != nil {
	// 	servicelog.ErrorLog().Print(err.Error())
	// 	writeError(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// defer rows.Close()

	// secs := []gomoex.Security{}
	// for rows.Next() {
	// 	s := gomoex.Security{}
	// 	err := rows.Scan(&s.Ticker, &s.LotSize, &s.ISIN, &s.Board, &s.Instrument)
	// 	if err != nil {
	// 		servicelog.ErrorLog().Print(err.Error())
	// 		writeError(w, err.Error(), http.StatusInternalServerError)
	// 		return
	// 	}
	// 	secs = append(secs, s)
	// }

	// resp, err := json.Marshal(secs)
	// if err != nil {
	// 	servicelog.ErrorLog().Print(err)
	// 	writeError(w, "Не удалось сериализовать данные", http.StatusInternalServerError)
	// 	return
	// }

	writeResponse(w, resp)
}

func (h *Handler) Coupons(w http.ResponseWriter, req *http.Request) {
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

func (h *Handler) Amortizations(w http.ResponseWriter, req *http.Request) {
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

func (h *Handler) BondIndicators(w http.ResponseWriter, req *http.Request) {
	isin := req.URL.Query().Get("isin")
	if isin == "" {
		writeError(w, "Не указан ISIN ценной бумаги", http.StatusBadRequest)
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
