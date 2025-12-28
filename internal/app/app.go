package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"simple-invest/internal/handlers"
	"simple-invest/internal/repository"
	"simple-invest/internal/securities"
)

type App struct {
	// TODO: logger
	db      *sql.DB
	server  *http.Server
	handler *handlers.Handler
}

func New() *App {
	db, err := dbPostgreSQL()
	if err != nil {
		log.Fatal(err)
	}

	repo := repository.NewPostgresRepo(db)
	service := securities.New(repo)
	handler := handlers.New(service)

	mux := http.NewServeMux()
	setupRoutes(mux, handler)

	// TODO: реализовать порт через конфигурацию
	port := "7540"
	app := &App{
		db: db,
		server: &http.Server{
			Handler: mux,
			Addr:    fmt.Sprintf(":%s", port),
		},
		handler: handler,
	}

	return app
}

func setupRoutes(mux *http.ServeMux, h *handlers.Handler) {
	mux.HandleFunc("/", h.DefaultHandle)
	mux.HandleFunc("GET /dividends", h.Dividends)
	mux.HandleFunc("GET /shares", h.Shares)
	mux.HandleFunc("GET /bonds", h.Bonds)
	mux.HandleFunc("GET /coupons", h.Coupons)
	mux.HandleFunc("GET /amortizations", h.Amortizations)
	mux.HandleFunc("GET /bondindicators", h.BondIndicators)
}

func (app *App) Run() error {
	if err := app.server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func (app *App) Stop(ctx context.Context) error {
	var errs []error
	if err := app.server.Shutdown(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := app.db.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// TODO: строка соединения входящим параметром
func dbPostgreSQL() (*sql.DB, error) {

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
