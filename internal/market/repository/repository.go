package repository

import (
	"context"
	"github.com/keruch/ton_telegram/internal/market/domain"
	"github.com/lib/pq"

	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/keruch/ton_telegram/pkg/logger"
)

type PostgresSQLPool struct {
	pool   *pgxpool.Pool
	logger *log.Logger
}

func NewPostgresSQLPool(url string, logger *log.Logger) (*PostgresSQLPool, error) {
	pool, err := pgxpool.Connect(context.Background(), url)
	if err != nil {
		return nil, err
	}

	if err = pool.Ping(context.Background()); err != nil {
		return nil, err
	}

	return &PostgresSQLPool{
		pool:   pool,
		logger: logger,
	}, nil
}

const insertUser = `INSERT INTO users(id, username)
VALUES ($1, $2)
ON CONFLICT (id) DO NOTHING`

func (p *PostgresSQLPool) AddUser(ctx context.Context, ID int64, username string) error {
	_, err := p.pool.Exec(ctx, insertUser, ID, username)
	return err
}

const getRating = `SELECT username, nft_count
FROM users
ORDER BY nft_count DESC
LIMIT $1`

func (p *PostgresSQLPool) GetRating(ctx context.Context, limit int) ([]domain.RatingRow, error) {
	rows, err := p.pool.Query(ctx, getRating, limit)
	if err != nil {
		return nil, err
	}
	var value []domain.RatingRow
	for rows.Next() {
		var username string
		var count int
		err = rows.Scan(&username, &count)
		if err != nil {
			return nil, err
		}
		value = append(value, domain.RatingRow{
			Username: username,
			Nft:      count,
		})
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return value, nil
}

const getIDs = `SELECT nft_id
FROM users
WHERE id = $1`

func (p *PostgresSQLPool) GetIDs(ctx context.Context, userID int64) ([]int, error) {
	rows, err := p.pool.Query(ctx, getIDs, userID)
	if err != nil {
		return nil, err
	}
	var values []int
	for rows.Next() {
		var ids pq.Int32Array
		err = rows.Scan(&ids)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			values = append(values, int(id))
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return values, nil
}
