package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/keruch/ton_masks_bot/pkg/logger"
)

var ErrorAlreadyRegistered = errors.New("user already registered")

type PostgresSQLPool struct {
	pool   *pgxpool.Pool
	logger *log.Logger

	mu                   *sync.Mutex
	registeredUsersCache map[int64]bool
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

const (
	insertUserDataTemplate = `insert into user_data 
(user_id, invited_by, points, openart, additional) 
values ('%v', '%v', '%v', '%v', '%v');`
	insertUserTemplate         = `insert into users (user_id, username) values ('%v', '%s');`
	updatePointsTemplate       = `update user_data set points = points + %d where user_id = '%v'`
	updateSubscriptionTemplate = `update user_data set %s = '%v' where user_id = '%v'`
	getFieldWithIDTemplate     = "select %s from user_data where user_id = '%v'"
	getInvitedByIDTemplate     = "select invited_by from user_data where user_id = '%v'"
	getUsernameByID            = "select username from users where user_id = '%v'"
)

func (p *PostgresSQLPool) AddUser(ctx context.Context, ID int64, username string, invitedID int64) error {
	ok, err := p.isUserAlreadyRegistered(ctx, ID)
	if err != nil {
		return err
	}
	if ok {
		return ErrorAlreadyRegistered
	}

	if _, err = p.pool.Exec(ctx, fmt.Sprintf(insertUserDataTemplate, ID, invitedID, 0, false, false)); err != nil {
		return err
	}
	if _, err = p.pool.Exec(ctx, fmt.Sprintf(insertUserTemplate, ID, username)); err != nil {
		return err
	}

	return nil
}

func (p *PostgresSQLPool) isUserAlreadyRegistered(ctx context.Context, ID int64) (bool, error) {
	if _, ok := p.registeredUsersCache[ID]; ok == true {
		return ok, nil
	}
	rows, err := p.pool.Query(ctx, fmt.Sprintf("select * from users where user_id = '%v'", ID))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		return true, nil
	}

	return false, nil
}

func (p *PostgresSQLPool) GetFieldForID(ctx context.Context, ID int64, field string) (interface{}, error) {
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

func (p *PostgresSQLPool) GetInvitedByID(ctx context.Context, ID int64) (int64, error) {
	getCommand := fmt.Sprintf(getInvitedByIDTemplate, ID)
	rows, err := p.pool.Query(ctx, getCommand)
	if err != nil {
		return 0, err
	}
	var value int64
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			return 0, err
		}
		break
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return 0, err
	}

	return value, nil
}

func (p *PostgresSQLPool) UpdatePoints(ctx context.Context, ID int64, points int) error {
	insertCommand := fmt.Sprintf(updatePointsTemplate, points, ID)
	_, err := p.pool.Exec(ctx, insertCommand)
	return err
}

func (p *PostgresSQLPool) UpdateSubscription(ctx context.Context, subscription string, ID int64, value bool) error {
	insertCommand := fmt.Sprintf(updateSubscriptionTemplate, subscription, value, ID)
	_, err := p.pool.Exec(ctx, insertCommand)
	return err
}

func (p *PostgresSQLPool) GetUsername(ctx context.Context, ID int64) (string, error) {
	getCommand := fmt.Sprintf(getUsernameByID, ID)
	rows, err := p.pool.Query(ctx, getCommand)
	if err != nil {
		return "", err
	}
	var value string
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			return "", err
		}
		break
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return "", err
	}

	return value, nil
}
