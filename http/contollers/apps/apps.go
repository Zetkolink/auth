package apps

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Zetkolink/auth/http/helpers"
	"github.com/Zetkolink/auth/models/apps"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

// Controller type represents HTTP-controller.
type Controller struct {
	models *ModelSet
}

// ModelSet type represents model set.
type ModelSet struct {
	Apps *apps.Model
}

type appRequest struct {
	*apps.App
}

type appResponse struct {
	*apps.App
}

type authCodeURLResponse struct {
	Url string `json:"url"`
}

// NewController method creates new controller instance.
func NewController(models ModelSet) *Controller {
	return &Controller{
		models: &models,
	}
}

// NewRouter method returns HTTP-router for controller.
func (c *Controller) NewRouter() chi.Router {
	r := chi.NewRouter()

	r.Patch("/{appID}/status/{status}", c.Create)

	r.Route("/{service}",
		func(r chi.Router) {
			r.Get("/", c.Get)
			r.Get("/{userID}", c.AuthCodeURL)
			r.Post("/", c.Create)
		},
	)

	return r
}

// Create handler creates new app.
func (c *Controller) Create(w http.ResponseWriter, r *http.Request) {
	payload := &appRequest{}
	err := render.Bind(r, payload)

	if err != nil {
		helpers.BadRequest(w, r, err)
		return
	}

	newApp := payload.App
	err = helpers.ConformStruct(newApp)

	if err != nil {
		helpers.InternalServerError(w, r, err)
		return
	}

	service := chi.URLParam(r, "service")

	if service == "" {
		helpers.NotFound(w, r, apps.ErrNotFound)
		return
	}

	newApp.Service = service

	errs := helpers.ValidateStruct(newApp, nil)

	if errs != nil {
		helpers.ValidationFailed(w, r, errs)
		return
	}

	id, err := c.models.Apps.Create(r.Context(), newApp)

	if err != nil {
		if err == apps.ErrExists {
			helpers.Conflict(w, r, err)
			return
		}

		helpers.InternalServerError(w, r, err)
		return
	}

	app, err := c.models.Apps.GetByID(r.Context(), id)

	if err != nil {
		helpers.InternalServerError(w, r, err)
		return
	} else if app == nil {
		helpers.NotFound(w, r, apps.ErrNotFound)
		return
	}

	w.WriteHeader(http.StatusCreated)
	render.Render(w, r, newAppResponse(app))
}

// SetStatus handler update app status.
func (c *Controller) SetStatus(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")

	if appID == "" {
		helpers.NotFound(w, r, apps.ErrNotFound)
		return
	}

	status := chi.URLParam(r, "status")

	if status == "" {
		helpers.NotFound(w, r, apps.ErrNotFound)
		return
	}

	app, err := c.models.Apps.SetStatus(r.Context(), appID, status)

	if err != nil {
		if err == apps.ErrExists {
			helpers.Conflict(w, r, err)
			return
		}

		helpers.InternalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	render.Render(w, r, newAppResponse(app))
}

// Get handler renders returns app.
func (c *Controller) Get(w http.ResponseWriter, r *http.Request) {
	service := chi.URLParam(r, "service")

	if service == "" {
		helpers.NotFound(w, r, apps.ErrNotFound)
		return
	}

	ctx := r.Context()
	app, err := c.models.Apps.GetByService(ctx, service)

	if err != nil {
		helpers.InternalServerError(w, r, err)
		return
	}

	if app == nil {
		helpers.NotFound(w, r, apps.ErrNotFound)
		return
	}

	render.Render(w, r, newAppResponse(app))
}

// AuthCodeURL handler renders returns auth code url.
func (c *Controller) AuthCodeURL(w http.ResponseWriter, r *http.Request) {
	service := chi.URLParam(r, "service")

	if service == "" {
		helpers.NotFound(w, r, apps.ErrNotFound)
		return
	}

	userID, err := strconv.Atoi(chi.URLParam(r, "userID"))

	if err != nil {
		helpers.BadRequest(w, r, err)
		return
	}

	ctx := r.Context()
	url, err := c.models.Apps.AuthCodeURL(ctx, service, userID)

	if err != nil {
		helpers.InternalServerError(w, r, err)
		return
	}

	if url == "" {
		helpers.NotFound(w, r, apps.ErrNotFound)
		return
	}

	render.Render(w, r, newAuthCodeURLResponse(url))
}

func (prs *appResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (ac *authCodeURLResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (prq *appRequest) Bind(_ *http.Request) error {
	if prq.App == nil {
		return errors.New("missing required App field")
	}

	return nil
}

func newAppResponse(app *apps.App) *appResponse {
	return &appResponse{
		App: app,
	}
}

func newAuthCodeURLResponse(url string) *authCodeURLResponse {
	return &authCodeURLResponse{
		Url: url,
	}
}
