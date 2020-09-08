package exchanges

import (
	"context"
	"database/sql"
)

type Model struct {
	db *sql.DB
}

type ModelConfig struct {
	Db *sql.DB
}

type Exchange struct {
	ID      string `json:"id"`
	Service string `json:"service"`
	UserID  int    `json:"user_id"`
}

func NewModel(config ModelConfig) (*Model, error) {
	m := &Model{db: config.Db}

	return m, nil
}

func (m *Model) Get(ctx context.Context, id string) (*Exchange, error) {
	var exchange Exchange

	err := m.db.QueryRowContext(ctx, `SELECT  
									"id", "service", "user_id"
									     FROM auth.exchanges
								WHERE id = $1`,
		id,
	).Scan(&exchange.ID, &exchange.Service, &exchange.UserID)

	if err != nil {
		return nil, err
	}

	return &exchange, nil
}

func (m *Model) Create(ctx context.Context, exchange *Exchange) (string, error) {
	_, err := m.db.ExecContext(ctx, `INSERT INTO auth.exchanges
									( "id", "service", "user_id")
								VALUES ($1, $2, $3)`,
		exchange.ID, exchange.Service, exchange.UserID,
	)

	if err != nil {
		return "", err
	}

	return exchange.ID, nil
}

func (m *Model) Delete(ctx context.Context, id string) error {
	_, err := m.db.ExecContext(ctx, `DELETE  
								FROM auth.exchanges
								WHERE id = $1`, id,
	)

	if err != nil {
		return err
	}

	return nil
}
