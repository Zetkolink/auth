package tokens

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Zetkolink/auth/models/apps"
	"github.com/Zetkolink/auth/models/exchanges"
	"golang.org/x/oauth2"
)

var (
	// ErrNotFound token not found.
	ErrNotFound = errors.New("token not found")
)

type Model struct {
	db        *sql.DB
	exchanges *exchanges.Model
	apps      *apps.Model
}

type ModelConfig struct {
	Db        *sql.DB
	Exchanges *exchanges.Model
	Apps      *apps.Model
}

type Token struct {
	*oauth2.Token
	UserID    int       `json:"user_id"`
	Service   string    `json:"service"`
	CreatedAt time.Time `json:"created_at"`
}

func NewModel(config ModelConfig) (*Model, error) {
	m := &Model{
		db:        config.Db,
		exchanges: config.Exchanges,
		apps:      config.Apps,
	}

	return m, nil
}

func (m *Model) Get(ctx context.Context, userID string, service string) (*Token, error) {
	token := Token{
		Token: &oauth2.Token{},
	}

	err := m.db.QueryRowContext(ctx, `SELECT  
									"user_id", "token_type","access_token", 
       								"expiry", "refresh_token",
       								"created_at", "service"
									     FROM auth.tokens
								WHERE user_id = $1 AND service = $2`,
		userID, service,
	).Scan(&token.UserID, &token.TokenType, &token.AccessToken,
		&token.Expiry, &token.RefreshToken,
		&token.CreatedAt, &token.Service,
	)

	if err != nil {
		return nil, err
	}

	return &token, nil
}

func (m *Model) Refresh(ctx context.Context, userID string, service string) (*Token, error) {
	token := Token{
		Token: &oauth2.Token{},
	}

	err := m.db.QueryRowContext(ctx, `SELECT  
									"user_id", "token_type","access_token", 
       								"expiry", "refresh_token",
       								"created_at", "service"
									     FROM auth.tokens
								WHERE user_id = $1 AND service = $2`,
		userID, service,
	).Scan(&token.UserID, &token.TokenType, &token.AccessToken,
		&token.Expiry, &token.RefreshToken,
		&token.CreatedAt, &token.Service,
	)

	if err != nil {
		return nil, err
	}

	conf, err := m.apps.GetConf(ctx, token.Service)

	if err != nil {
		return nil, err
	}

	ts := conf.TokenSource(ctx, token.Token)
	newToken, err := ts.Token()

	if err != nil {
		return nil, err
	}

	_, err = m.db.ExecContext(ctx, `UPDATE auth.tokens SET
									"access_token" = $2,
                       				"refresh_token" = $3,
       								"expiry" = $4,
       								"created_at" = $5
								WHERE user_id = $1`,
		userID, newToken.AccessToken, newToken.RefreshToken,
		newToken.Expiry, time.Now(),
	)

	if err != nil {
		return nil, err
	}

	return &token, nil
}

func (m *Model) Create(ctx context.Context, code string, exchangeID string) (int, error) {
	exchange, err := m.exchanges.Get(ctx, exchangeID)

	if err != nil {
		return 0, err
	}

	conf, err := m.apps.GetConf(ctx, exchange.Service)

	if err != nil {
		return 0, err
	}

	tk, err := conf.Exchange(ctx, code)

	if err != nil {
		return 0, err
	}

	_ = m.exchanges.Delete(ctx, exchangeID)

	_, err = m.db.ExecContext(ctx, `INSERT INTO auth.tokens
									( "user_id", "token_type","access_token", 
       								"expiry", "refresh_token",
       								"created_at", "service" )
								VALUES ($1, $2, $3, $4, $5, $6, $7) 
								ON CONFLICT (user_id, service) DO UPDATE 
								SET access_token = excluded.access_token,
								refresh_token = excluded.refresh_token,
								expiry = excluded.expiry,
								created_at = excluded.created_at`,
		exchange.UserID, tk.TokenType, tk.AccessToken,
		tk.Expiry, tk.RefreshToken,
		time.Now(), exchange.Service,
	)

	if err != nil {
		return 0, err
	}

	return exchange.UserID, nil
}
