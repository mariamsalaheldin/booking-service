package repository

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Postgres struct {
	DB *sql.DB
}

func NewPostgres(databaseURL string) (*Postgres, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return &Postgres{
		DB: db,
	}, nil
}

func (p *Postgres) Close() error {
	return p.DB.Close()
}

func (p *Postgres) Health(ctx context.Context) error {
	return p.DB.PingContext(ctx)
}
