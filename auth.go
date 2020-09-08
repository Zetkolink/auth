package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Zetkolink/auth/models/apps"
	"github.com/Zetkolink/auth/models/exchanges"
	"github.com/Zetkolink/auth/models/tokens"
	_ "github.com/lib/pq"
)

type auth struct {
	db         *sql.DB
	httpServer *http.Server
	models     modelSet
	wg         sync.WaitGroup
}

type modelSet struct {
	Exchanges *exchanges.Model
	Apps      *apps.Model
	Tokens    *tokens.Model
}

type config struct {
	Db   dbConfig
	Http httpConfig
}

type dbConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

type httpConfig struct {
	Bind              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}

func newAuth() (*auth, error) {
	db, err := sql.Open("postgres", cfg.Db.GetConn())

	if err != nil {
		return nil, err
	}

	err = db.Ping()

	if err != nil {
		return nil, err
	}

	exchangesModel, err := exchanges.NewModel(
		exchanges.ModelConfig{Db: db},
	)

	appsModel, err := apps.NewModel(
		apps.ModelConfig{
			Db:        db,
			Exchanges: exchangesModel,
		},
	)

	tokensModel, err := tokens.NewModel(
		tokens.ModelConfig{
			Db:        db,
			Exchanges: exchangesModel,
			Apps:      appsModel,
		},
	)

	if err != nil {
		return nil, err
	}

	a := auth{
		db: db,
		models: modelSet{
			Exchanges: exchangesModel,
			Apps:      appsModel,
			Tokens:    tokensModel,
		},
	}

	err = a.setupHTTPServer(cfg.Http)

	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (s *auth) Run() error {
	s.runHTTPServer()

	return nil
}

func (s *auth) runHTTPServer() {
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		err := s.httpServer.ListenAndServe()

		if err != http.ErrServerClosed {
			s.Stop()
		}
	}()
}

func (s *auth) Stop() {
	err := s.httpServer.Shutdown(context.Background())

	if err != nil {
		log.Println(err)
	}

	s.wg.Wait()
}

func (d *dbConfig) GetConn() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Db.Host, cfg.Db.Port, cfg.Db.User, cfg.Db.Password,
		cfg.Db.Database,
	)
}
