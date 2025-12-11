package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreSQL struct {
	Client *pgxpool.Pool
}

func NewPostgres(dsn string) (PostgreSQL, error) {
	var pg PostgreSQL

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return pg, err
	}

	cfg.MaxConns = 100

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return pg, err
	}

	if err := pool.Ping(ctx); err != nil {
		return pg, err
	}

	pg.Client = pool
	return pg, nil
}
