package repository

import (
	"database/sql"

	"github.com/WLM1ke/gomoex"
	_ "github.com/lib/pq"
)

type Repository interface {
	GetShares() ([]gomoex.Security, error)
	GetBonds() ([]gomoex.Security, error)
	UpdateShares([]gomoex.Security) (int, error)
	UpdateBonds([]gomoex.Security) (int, error)
}

// Реализация PostgreSQL
type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) GetShares() ([]gomoex.Security, error) {
	rows, err := r.db.Query("SELECT ticker, lotsize, isin, board, instrument FROM securities")

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	secs := []gomoex.Security{}
	for rows.Next() {
		s := gomoex.Security{}
		err := rows.Scan(&s.Ticker, &s.LotSize, &s.ISIN, &s.Board, &s.Instrument)
		if err != nil {
			return nil, err
		}
		secs = append(secs, s)
	}

	return secs, err
}

func (r *PostgresRepo) GetBonds() ([]gomoex.Security, error) {
	rows, err := r.db.Query("SELECT ticker, lotsize, isin, board, instrument FROM securities")

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	secs := []gomoex.Security{}
	for rows.Next() {
		s := gomoex.Security{}
		err := rows.Scan(&s.Ticker, &s.LotSize, &s.ISIN, &s.Board, &s.Instrument)
		if err != nil {
			return nil, err
		}
		secs = append(secs, s)
	}

	return secs, err
}

func (r *PostgresRepo) UpdateShares(secs []gomoex.Security) (int, error) {
	n, err := updateSecurities(r, secs)
	return n, err
}

func (r *PostgresRepo) UpdateBonds(secs []gomoex.Security) (int, error) {
	n, err := updateSecurities(r, secs)
	return n, err
}

func updateSecurities(r *PostgresRepo, secs []gomoex.Security) (int, error) {
	// securityType string, - добавить входной параметр в функцию для фильтрации ЦБ
	existing := make(map[string]bool)
	rows, err := r.db.Query("SELECT isin FROM securities")
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var isin string
		err = rows.Scan(&isin)
		if err != nil {
			return 0, err
		}
		existing[isin] = false
	}

	updated := 0
	for _, s := range secs {
		_, ok := existing[s.ISIN]
		if ok {
			// тут обновление
			_, err := r.db.Exec(`
			UPDATE securities
			SET ticker = $2,
				lotsize = $3,
				board = $4,
				sectype = $5,
				instrument = $6
			WHERE isin = $1;`, s.ISIN, s.Ticker, s.LotSize, s.Board, s.Type, s.Instrument)
			if err != nil {
				return updated, err
			}
		} else {
			// тут создание
			_, err := r.db.Exec(`
				INSERT INTO securities (isin, ticker, lotsize, board, sectype, instrument)
				VALUES ($1, $2, $3, $4, $5, $6)`, s.ISIN, s.Ticker, s.LotSize, s.Board, s.Type, s.Instrument)
			if err != nil {
				return updated, err
			}
		}
		updated++
	}
	return updated, nil
}
