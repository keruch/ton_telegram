package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/keruch/the_open_art_ton_bot/pkg/logger"
)

var ErrorAlreadyRegistered = errors.New("user already registered")

type PostgreSQLPool struct {
	pool   *pgxpool.Pool
	logger *log.Logger
}

func NewPostgreSQLPool(url string, logger *log.Logger) (*PostgreSQLPool, error) {
	pool, err := pgxpool.Connect(context.Background(), url)
	if err != nil {
		return nil, err
	}

	if err = pool.Ping(context.Background()); err != nil {
		return nil, err
	}

	return &PostgreSQLPool{
		pool:   pool,
		logger: logger,
	}, nil
}

const (
	insertUserTemplate = `insert into users 
(user_id, username, invited_by, chat_id, points, openart, planets) 
values ('%v', '%s', '%v', '%v', '%v', '%v', '%v');`
	updatePointsTemplate   = `update users set points = points + %d where user_id = '%v'`
	updateOpenArtTemplate  = `update users set %s = '%v' where user_id = '%v'`
	getFieldWithIDTemplate = "select %s from users where user_id = '%d'"
)

func (p *PostgreSQLPool) AddUser(ctx context.Context, ID int64, username string, invitedID int64, chatID int64) error {
	rows, err := p.pool.Query(ctx, fmt.Sprintf("select * from users where user_id = '%d'", ID))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		return ErrorAlreadyRegistered
	}

	insertCommand := fmt.Sprintf(insertUserTemplate, ID, username, invitedID, chatID, 0, false, false)
	_, err = p.pool.Exec(ctx, insertCommand)
	return err
}

func (p *PostgreSQLPool) GetFieldForID(ctx context.Context, ID int64, field string) (interface{}, error) {
	p.logger.Tracef("Get points with args %d", ID)
	getCommand := fmt.Sprintf(getFieldWithIDTemplate, field, ID)
	rows, err := p.pool.Query(ctx, getCommand)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var value interface{}
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			return 0, err
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return value, nil
}

func (p *PostgreSQLPool) UpdatePoints(ctx context.Context, ID int64, points int) error {
	insertCommand := fmt.Sprintf(updatePointsTemplate, points, ID)
	_, err := p.pool.Exec(ctx, insertCommand)
	return err
}

func (p *PostgreSQLPool) UpdateSubscription(ctx context.Context, subscription string, ID int64, value bool) error {
	insertCommand := fmt.Sprintf(updateOpenArtTemplate, subscription, value, ID)
	_, err := p.pool.Exec(ctx, insertCommand)
	return err
}
