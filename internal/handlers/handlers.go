package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"simple-invest/internal/securities"
)

const (
	msgSerializationFailed   = "Serialization data failed"
	msgMoexGettingDataFailed = "Cannot get data from MOEX"
	msgEmptyID               = "Share ID cannot be empty"
)

var errEmtyID = errors.New(msgEmptyID)

type Handler struct {
	service *securities.SecuritiesService
}

func New(service *securities.SecuritiesService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) DefaultHandle(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("service working"))
}

func (h *Handler) Shares(w http.ResponseWriter, req *http.Request) {
	update := req.URL.Query().Get("update")
	if update == "yes" {
		if err := h.service.DownloadShares(); err != nil {
			log.Print(err.Error())
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
	}

	secs, err := h.service.Shares()
	if err != nil {
		log.Print(err.Error())
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(secs)
	if err != nil {
		log.Print(err)
		writeError(w, msgSerializationFailed, http.StatusInternalServerError)
		return
	}

	writeResponse(w, resp)
}

func (h *Handler) Dividends(w http.ResponseWriter, req *http.Request) {
	isin := req.URL.Query().Get("ticker")
	if isin == "" {
		log.Print(msgEmptyID)
		writeError(w, msgEmptyID, http.StatusBadRequest)
		return
	}

	divs, err := h.service.Dividends(context.Background(), isin)
	if err != nil {
		log.Print(err)
		writeError(w, msgMoexGettingDataFailed, http.StatusServiceUnavailable)
		return
	}

	resp, err := json.Marshal(divs)
	if err != nil {
		log.Print(err)
		writeError(w, msgSerializationFailed, http.StatusInternalServerError)
		return
	}

	writeResponse(w, resp)

}

func (h *Handler) Bonds(w http.ResponseWriter, req *http.Request) {
	update := req.URL.Query().Get("update")
	if update == "yes" {
		if err := h.service.DownloadBonds(); err != nil {
			log.Print(err.Error())
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
	}

	secs, err := h.service.Bonds()
	if err != nil {
		log.Print(err.Error())
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(secs)
	if err != nil {
		log.Print(err)
		writeError(w, msgSerializationFailed, http.StatusInternalServerError)
		return
	}

	writeResponse(w, resp)
}

func (h *Handler) Coupons(w http.ResponseWriter, req *http.Request) {
	isin := req.URL.Query().Get("isin")
	if isin == "" {
		log.Print(msgEmptyID)
		writeError(w, msgEmptyID, http.StatusBadRequest)
		return
	}

	coupons, err := h.service.Coupons(isin)
	if err != nil {
		log.Print(err)
		writeError(w, msgMoexGettingDataFailed, http.StatusServiceUnavailable)
		return
	}

	resp, err := json.Marshal(coupons)
	if err != nil {
		log.Print(err)
		writeError(w, msgSerializationFailed, http.StatusInternalServerError)
		return
	}

	writeResponse(w, resp)
}

func (h *Handler) Amortizations(w http.ResponseWriter, req *http.Request) {
	isin := req.URL.Query().Get("isin")
	if isin == "" {
		log.Print(msgEmptyID)
		writeError(w, msgEmptyID, http.StatusBadRequest)
		return
	}

	amortizations, err := h.service.Amortizations(isin)
	if err != nil {
		log.Print(err)
		writeError(w, msgMoexGettingDataFailed, http.StatusServiceUnavailable)
		return
	}

	resp, err := json.Marshal(amortizations)
	if err != nil {
		log.Print(err)
		writeError(w, msgSerializationFailed, http.StatusInternalServerError)
		return
	}

	writeResponse(w, resp)
}

func (h *Handler) BondIndicators(w http.ResponseWriter, req *http.Request) {
	isin := req.URL.Query().Get("isin")
	if isin == "" {
		log.Print(msgEmptyID)
		writeError(w, msgEmptyID, http.StatusBadRequest)
		return
	}

	bondIndicators, err := h.service.BondIndicators(isin)
	if err != nil {
		log.Print(err)
		writeError(w, msgMoexGettingDataFailed, http.StatusServiceUnavailable)
		return
	}

	resp, err := json.Marshal(bondIndicators)
	if err != nil {
		log.Print(err)
		writeError(w, msgSerializationFailed, http.StatusInternalServerError)
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
