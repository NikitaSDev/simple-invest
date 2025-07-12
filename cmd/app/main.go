package main

import (
	"net/http"
	"simple-invest/internal/database"
	"simple-invest/internal/handlers"
	"simple-invest/internal/servicelog"
	// "github.com/nikitasdev/simple-invest/internal/database"
)

func main() {
	defer servicelog.InfoLog().Print("Сервер остановлен")
	defer database.DB().Close()

	servicelog.InfoLog().Print("Подключение к базе данных установлено")

	mux := http.NewServeMux()
	mux.HandleFunc("/", handlers.DefaultHandle)
	mux.HandleFunc("/dividends", handlers.Dividends)
	mux.HandleFunc("/shares", handlers.Shares)
	mux.HandleFunc("/bonds", handlers.Bonds)
	mux.HandleFunc("/coupons", handlers.Coupons)
	mux.HandleFunc("/amortizations", handlers.Amortizations)
	mux.HandleFunc("/bondindicators", handlers.BondIndicators)

	port := ":7540"
	servicelog.InfoLog().Print("Запуск сервера")
	if err := http.ListenAndServe(port, mux); err != nil {
		servicelog.ErrorLog().Printf("ошибка запуска сервера: %s", err.Error())
	}
}
