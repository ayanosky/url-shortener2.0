package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

var _ Storage = (*Postgres)(nil)

func (p *Postgres) Save(ctx context.Context, code, url string) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO links (code, url) VALUES ($1, $2)`,
		code, url,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		// 23505 = unique_violation: такой code уже есть
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return fmt.Errorf("insert link: %w", err)
	}
	return nil
}

func (p *Postgres) Get(ctx context.Context, code string) (Link, error) {
	var l Link

	err := p.pool.QueryRow(ctx,
		`SELECT code, url, clicks FROM links WHERE code = $1`,
		code,
	).Scan(&l.Code, &l.URL, &l.Clicks)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Link{}, ErrNotFound
		}
		return Link{}, fmt.Errorf("select link: %w", err)
	}

	return l, nil
}

func (p *Postgres) IncrementClicks(ctx context.Context, code string) error {
	tag, err := p.pool.Exec(ctx, `UPDATE links SET clicks = clicks + 1 WHERE code = $1`, code)
	if err != nil {
		return fmt.Errorf("update link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
