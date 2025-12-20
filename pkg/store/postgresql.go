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

	cfg.MaxConns = 5             // Максимум 5 подключений на инстанс
	cfg.MinConns = 1             // Минимум 1 подключение
	cfg.MaxConnLifetime = 3 * 60 // 3 минуты время жизни
	cfg.MaxConnIdleTime = 15     // 15 секунд максимальное время простоя
	cfg.HealthCheckPeriod = 30   // Проверка каждые 30 секунд
	cfg.MaxConnIdleTime = 60     // 60 секунд максимальное время простоя

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
