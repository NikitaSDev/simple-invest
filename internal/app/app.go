package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"simple-invest/internal/handlers"
	"simple-invest/internal/repository"
	"simple-invest/internal/securities"
)

type App struct {
	log     *slog.Logger
	db      *sql.DB
	server  *http.Server
	handler *handlers.Handler
}

func New(log *slog.Logger, storagePath, port string) *App {
	db, err := dbPostgreSQL(storagePath)
	if err != nil {
		panic(err)
	}

	repo := repository.NewPostgresRepo(db)
	service := securities.New(repo)
	handler := handlers.New(service)

	mux := http.NewServeMux()
	setupRoutes(mux, handler)

	app := &App{
		log: log,
		db:  db,
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
	mux.HandleFunc("GET /shares", h.Shares)
	mux.HandleFunc("GET /bonds", h.Bonds)
	mux.HandleFunc("GET /dividends", h.Dividends)
	mux.HandleFunc("GET /coupons", h.Coupons)
	mux.HandleFunc("GET /amortizations", h.Amortizations)
	mux.HandleFunc("GET /bondindicators", h.BondIndicators)
}

func (app *App) MustRun() {
	app.log.Info("server started")
	if err := app.server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			app.log.Error(err.Error())
			panic(err)
		}
	}
}

func (app *App) Stop(ctx context.Context) {
	if err := app.server.Shutdown(ctx); err != nil {
		app.log.Error(err.Error())
	}

	if err := app.db.Close(); err != nil {
		app.log.Error(err.Error())
	}
}

func dbPostgreSQL(storagePath string) (*sql.DB, error) {
	db, err := sql.Open("postgres", storagePath)
	if err != nil {
		return nil, err
	}
	return db, nil
}
