package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Zetkolink/auth/http/contollers/apps"
	"github.com/Zetkolink/auth/http/contollers/tokens"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"subs/http/helpers"
)

func (s *auth) setupHTTPServer(config httpConfig) error {
	config.ReadTimeout *= time.Second
	config.ReadHeaderTimeout *= time.Second
	config.WriteTimeout *= time.Second
	config.IdleTimeout *= time.Second

	apiVersion := "v1"

	r := chi.NewRouter()
	r.Use(middleware.WithValue(helpers.APIVersionContextKey, apiVersion))
	r.Use(middleware.StripSlashes)
	r.Use(middleware.Recoverer)

	r.Route(
		fmt.Sprintf("%s/%s", helpers.APIPathSuffix, apiVersion),

		func(r chi.Router) {
			r.Group(
				func(r chi.Router) {
					appsController := apps.NewController(
						apps.ModelSet{
							Apps: s.models.Apps,
						},
					)

					r.Mount(
						"/apps",
						appsController.NewRouter(),
					)

					tokensController := tokens.NewController(
						tokens.ModelSet{
							Tokens: s.models.Tokens,
						},
					)

					r.Mount(
						"/tokens",
						tokensController.NewRouter(),
					)
				},
			)
		},
	)

	s.httpServer = &http.Server{
		Addr:              config.Bind,
		Handler:           r,
		ReadTimeout:       config.ReadTimeout,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		MaxHeaderBytes:    config.MaxHeaderBytes,
	}

	return nil
}
