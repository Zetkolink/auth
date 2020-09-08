package apps

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Zetkolink/auth/http/helpers"
	"github.com/Zetkolink/auth/models/exchanges"
	"github.com/lib/pq"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/mailru"
	"golang.org/x/oauth2/vk"
	"golang.org/x/oauth2/yandex"
)

const (
	StatusEnable  = "enable"
	StatusDisable = "disable"

	Google = "google"
	Yandex = "yandex"
	Mail   = "mail"
	VK     = "vk"
)

var (
	// ErrNotFound app not found.
	ErrNotFound = errors.New("app not found")

	// ErrExists app exists.
	ErrExists = errors.New("app exists")

	// ErrStatus app status unavailable.
	ErrStatus = errors.New("app status unavailable")

	// ErrService app status unavailable.
	ErrService = errors.New("app service unavailable")

	// TODO rework
	scopes = map[string][]string{
		Yandex: {"mail:imap_ro"},
		Google: {"https://www.googleapis.com/github.com/Zetkolink/auth/gmail.addons.current.message.readonly"},
	}
)

type Model struct {
	db        *sql.DB
	exchanges *exchanges.Model
}

type ModelConfig struct {
	Db        *sql.DB
	Exchanges *exchanges.Model
}

type App struct {
	ID          string     `json:"id"`
	Service     string     `json:"service"`
	Password    string     `json:"password"`
	CallbackURL string     `json:"callback_URL"`
	Expiry      *time.Time `json:"expiry"`
	CreatedAt   *time.Time `json:"created_at"`
	Status      string     `json:"status"`
}

func NewModel(config ModelConfig) (*Model, error) {
	m := &Model{
		db:        config.Db,
		exchanges: config.Exchanges,
	}

	return m, nil
}

func (m *Model) GetByID(ctx context.Context, id string) (*App, error) {
	var app App

	err := m.db.QueryRowContext(ctx, `SELECT  
									"id", "service","password", 
       								"callback_URL", "expiry",
       								"created_at"
									     FROM auth.apps
								WHERE id = $1`,
		id,
	).Scan(&app.ID, &app.Service, &app.Password, &app.CallbackURL,
		&app.Expiry, &app.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &app, nil
}

func (m *Model) GetByService(ctx context.Context, service string) (*App, error) {
	var app App

	err := m.db.QueryRowContext(ctx, `SELECT  
									"id", "service","password", 
       								"callback_URL", "expiry",
       								"created_at"
									     FROM auth.apps
								WHERE service = $1 AND status = $2`,
		service, StatusEnable,
	).Scan(&app.ID, &app.Service, &app.Password, &app.CallbackURL,
		&app.Expiry, &app.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &app, nil
}

func (m *Model) GetConf(ctx context.Context, service string) (*oauth2.Config, error) {
	var app App

	err := m.db.QueryRowContext(ctx, `SELECT  
									"id", "service","password", 
       								"callback_URL", "expiry",
       								"created_at"
									     FROM auth.apps
								WHERE service = $1 AND status = $2`,
		service, StatusEnable,
	).Scan(&app.ID, &app.Service, &app.Password, &app.CallbackURL,
		&app.Expiry, &app.CreatedAt)

	if err != nil {
		return nil, err
	}

	conf := &oauth2.Config{
		ClientID:     app.ID,
		ClientSecret: app.Password,
		Scopes:       scopes[app.Service],
		RedirectURL:  app.CallbackURL,
	}

	switch app.Service {
	case Yandex:
		conf.Endpoint = yandex.Endpoint
	case Google:
		conf.Endpoint = google.Endpoint
	case Mail:
		conf.Endpoint = mailru.Endpoint
	case VK:
		conf.Endpoint = vk.Endpoint
	default:
		return nil, ErrService
	}

	return conf, nil
}

func (m *Model) AuthCodeURL(ctx context.Context, service string, userID int) (string, error) {
	conf, err := m.GetConf(ctx, service)

	if err != nil {
		return "", err
	}

	var exchange exchanges.Exchange

	exchange.Service = service
	exchange.UserID = userID
	exchange.ID, err = helpers.RandomStr(32)

	if err != nil {
		return "", err
	}

	_, err = m.exchanges.Create(ctx, &exchange)

	if err != nil {
		return "", err
	}

	return conf.AuthCodeURL(exchange.ID), nil
}

func (m *Model) SetStatus(ctx context.Context, id string, status string) (*App, error) {
	var app App

	if status != StatusDisable && status != StatusEnable {
		return nil, ErrStatus
	}

	err := m.db.QueryRowContext(ctx, `UPDATE auth.apps 
								SET status = $2
								WHERE id = $1`,
		id, status,
	).Scan()

	if err != nil {
		return nil, err
	}

	return &app, nil
}

func (m *Model) Create(ctx context.Context, app *App) (string, error) {
	_, err := m.db.ExecContext(ctx, `INSERT INTO auth.apps
									( "id", "service","password", 
									 "callback_URL", "expiry",
									 "created_at", "status")
								VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		app.ID, app.Service, app.Password, app.CallbackURL,
		app.Expiry, time.Now(), app.Status,
	)

	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok {
			if pgErr.Code == "23505" {
				return "", ErrExists
			}
		}

		return "", err
	}

	return app.ID, nil
}
