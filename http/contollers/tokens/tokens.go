package tokens

import (
	"errors"
	"net/http"

	"github.com/Zetkolink/auth/http/helpers"
	"github.com/Zetkolink/auth/models/tokens"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

// Controller type represents HTTP-controller.
type Controller struct {
	models *ModelSet
}

// ModelSet type represents model set.
type ModelSet struct {
	Tokens *tokens.Model
}

type tokenResponse struct {
	*tokens.Token
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

	r.Get("/", c.Create)
	r.Get("/{userID}/{service}", c.Get)
	r.Put("/{userID}/{service}", c.Refresh)

	return r
}

// Create handler creates new token.
func (c *Controller) Create(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")

	if code == "" {
		helpers.BadRequest(w, r, errors.New("code not specified"))
		return
	}

	state := r.FormValue("state")

	if state == "" {
		helpers.BadRequest(w, r, errors.New("state not specified"))
		return
	}

	_, err := c.models.Tokens.Create(r.Context(), code, state)

	if err != nil {
		helpers.InternalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	render.Respond(w, r, "")
}

// Get handler renders returns token.
func (c *Controller) Get(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")

	if userID == "" {
		helpers.NotFound(w, r, tokens.ErrNotFound)
		return
	}

	service := chi.URLParam(r, "service")

	if service == "" {
		helpers.NotFound(w, r, tokens.ErrNotFound)
		return
	}

	ctx := r.Context()
	token, err := c.models.Tokens.Get(ctx, userID, service)

	if err != nil {
		helpers.InternalServerError(w, r, err)
		return
	}

	if token == nil {
		helpers.NotFound(w, r, tokens.ErrNotFound)
		return
	}

	render.Render(w, r, newTokenResponse(token))
}

// Refresh handler refresh token.
func (c *Controller) Refresh(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")

	if userID == "" {
		helpers.NotFound(w, r, tokens.ErrNotFound)
		return
	}

	service := chi.URLParam(r, "service")

	if service == "" {
		helpers.NotFound(w, r, tokens.ErrNotFound)
		return
	}

	ctx := r.Context()
	token, err := c.models.Tokens.Refresh(ctx, userID, service)

	if err != nil {
		helpers.InternalServerError(w, r, err)
		return
	}

	if token == nil {
		helpers.NotFound(w, r, tokens.ErrNotFound)
		return
	}

	render.Render(w, r, newTokenResponse(token))
}

func (prs *tokenResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func newTokenResponse(token *tokens.Token) *tokenResponse {
	return &tokenResponse{
		Token: token,
	}
}
