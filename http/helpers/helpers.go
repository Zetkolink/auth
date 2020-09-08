package helpers

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/render"
	"gopkg.in/go-playground/mold.v2/modifiers"
	"gopkg.in/go-playground/validator.v9"
)

const (
	// APIPathSuffix is the path suffix for API endpoint URL.
	APIPathSuffix = "/api"

	// RFC339Short short version of time.RFC339.
	RFC339Short = "2006-01-02"

	defaultSchema = "http"
	defaultPage   = 1
	maxPerPage    = 1000

	chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	// APIVersionContextKey is context key for API version.
	APIVersionContextKey = &contextKey{"apiVersion"}

	// PaginatorContextKey is context key for paginator.
	PaginatorContextKey = &contextKey{"paginator"}

	// UserRoleContextKey is context key for role.
	UserRoleContextKey = &contextKey{"userRole"}
)

var (
	conform = modifiers.New()

	validate = validator.New()

	protectedHTTPMethods = map[string]struct{}{
		http.MethodPost:   {},
		http.MethodPut:    {},
		http.MethodDelete: {},
	}
)

// Paginator type represents paginator.
type Paginator struct {
	Total   int
	PerPage int
	Page    int
}

// ErrorResponse type represents error response.
type ErrorResponse struct {
	StatusCode int    `json:"-"`
	Error      string `json:"error"`
}

// ValidationErrors type represents validation errors.
type ValidationErrors map[string]string

// ValidationErrorsResponse type represents validation errors response instance.
type ValidationErrorsResponse struct {
	Errors ValidationErrors `json:"errors"`
}

type paginateForm struct {
	Page    int
	PerPage int
}

type contextKey struct {
	name string
}

func init() {
	validate.RegisterTagNameFunc(
		func(field reflect.StructField) string {
			name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]

			if name == "-" {
				return ""
			}

			return name
		},
	)
}

// ConformStruct conform structure.
func ConformStruct(s interface{}) error {
	return conform.Struct(context.Background(), s)
}

// AccessController is a middleware for checking access privileges.
func AccessController(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		roleMap := make(map[string]struct{})

		for _, role := range roles {
			roleMap[role] = struct{}{}
		}

		handler := func(w http.ResponseWriter, r *http.Request) {
			role := GetUserRole(r)
			allowed := false

			if len(roleMap) > 0 {
				if _, ok := roleMap[role]; ok {
					allowed = true
				}
			} else if _, ok := protectedHTTPMethods[r.Method]; ok {
				if role == "admin" {
					allowed = true
				}
			} else {
				allowed = true
			}

			if !allowed {
				Forbidden(w, r)
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(handler)
	}
}

// GetUserRole method returns user role.
func GetUserRole(r *http.Request) string {
	if role, ok := r.Context().Value(UserRoleContextKey).(string); ok {
		return role
	}

	return ""
}

// Paginate is a middleware for pagination.
func Paginate(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var form paginateForm
			errs := decodePaginateForm(r, &form)

			if errs != nil {
				ValidationFailed(w, r, errs)
				return
			}

			ctx := context.WithValue(
				r.Context(),
				PaginatorContextKey,

				&Paginator{
					PerPage: form.PerPage,
					Page:    form.Page,
				},
			)

			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		},
	)
}

// ValidateStruct method validates structure.
func ValidateStruct(s interface{}, ffn validator.FilterFunc) ValidationErrors {
	if ffn == nil {
		ffn = func(ns []byte) bool {
			return false
		}
	}

	vldErr := validate.StructFiltered(s, ffn)

	if vldErr != nil {
		var errs = make(ValidationErrors)

		for _, err := range vldErr.(validator.ValidationErrors) {
			var errStr string
			tag := err.Tag()
			tagParam := err.Param()

			switch tag {
			case "required":
				errStr = "value is required"
			case "gt":
				if tagParam == "0" {
					if err.Kind() == reflect.Map {
						errStr = "empty map specified"
					} else if err.Kind() == reflect.Slice {
						errStr = "empty list specified"
					}
				}
			}

			if errStr == "" {
				errStr = "invalid value"
			}

			errs[err.Field()] = errStr
		}

		return errs
	}

	return nil
}

// NewErrorResponse method creates new error response instance.
func NewErrorResponse(statusCode int, err error) *ErrorResponse {
	return &ErrorResponse{
		StatusCode: statusCode,
		Error:      err.Error(),
	}
}

// Render method is a rendering hook.
func (e *ErrorResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.StatusCode)
	return nil
}

// NewValidationErrorsResponse method creates new error response instance.
func NewValidationErrorsResponse(errs ValidationErrors) *ValidationErrorsResponse {
	return &ValidationErrorsResponse{
		Errors: errs,
	}
}

// Render method is a rendering hook.
func (e *ValidationErrorsResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	render.Status(r, http.StatusUnprocessableEntity)
	return nil
}

// ValidationFailed method renders validation errors.
func ValidationFailed(w http.ResponseWriter, r *http.Request,
	errs ValidationErrors) {

	render.Render(w, r, NewValidationErrorsResponse(errs))
}

// NotFound method renders error with status code 404.
func NotFound(w http.ResponseWriter, r *http.Request, err error) {
	render.Render(w, r, NewErrorResponse(http.StatusNotFound, err))
}

// Conflict method renders error with status code 404.
func Conflict(w http.ResponseWriter, r *http.Request, err error) {
	render.Render(w, r, NewErrorResponse(http.StatusConflict, err))
}

// BadRequest method renders error with status code 400
func BadRequest(w http.ResponseWriter, r *http.Request, err error) {
	render.Render(w, r, NewErrorResponse(http.StatusBadRequest, err))
}

// Unauthorized method renders error with status code 401
func Unauthorized(w http.ResponseWriter, r *http.Request, _ error) {
	render.Render(w, r, NewErrorResponse(http.StatusUnauthorized,
		errors.New("401 Unauthorized")))
}

// Forbidden method renders error with status code 403
func Forbidden(w http.ResponseWriter, r *http.Request) {
	render.Render(w, r, NewErrorResponse(http.StatusForbidden,
		errors.New("403 Forbidden")))
}

// InternalServerError method renders error with status code 500.
func InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	render.Render(w, r, NewErrorResponse(http.StatusInternalServerError,
		errors.New(err.Error())))
}

// ParseDate function is a helper for parsing input date.
func ParseDate(s string) (time.Time, error) {
	date, err := time.Parse(RFC339Short, s)

	if err != nil {
		date, err = time.Parse(time.RFC3339, s)
	}

	if err != nil {
		return time.Time{}, err
	}

	return date, nil
}

func decodePaginateForm(r *http.Request, form *paginateForm) ValidationErrors {
	var errs = make(ValidationErrors)

	var err error

	page := r.FormValue("page")

	if page != "" {
		form.Page, err = strconv.Atoi(page)

		if err != nil || form.Page < 0 {
			errs["page"] = "invalid value specified"
		}
	}

	if form.Page == 0 {
		form.Page = defaultPage
	}

	perPage := r.FormValue("per_page")

	if perPage != "" {
		form.PerPage, err = strconv.Atoi(perPage)

		if err != nil || form.PerPage < 0 {
			errs["per_page"] = "invalid value specified"
		}
	}

	if form.PerPage > 0 &&
		form.PerPage > maxPerPage {

		form.PerPage = maxPerPage
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

// Skip method calculates and returns skip value.
func (p *Paginator) Skip() int {
	return (p.Page - 1) * p.PerPage
}

// Limit method calculates and returns limit value.
func (p *Paginator) Limit() int {
	return p.PerPage
}

// SetHeaders method sets paginator headers.
func (p *Paginator) SetHeaders(w http.ResponseWriter, _ *http.Request) {
	totalPages := p.Total / p.PerPage
	if p.Total%p.PerPage > 0 {
		totalPages++
	}

	headers := w.Header()
	headers.Add("X-Total", strconv.Itoa(p.Total))
	headers.Add("X-Total-Pages", strconv.Itoa(totalPages))
	headers.Add("X-Per-Page", strconv.Itoa(p.PerPage))
	headers.Add("X-Page", strconv.Itoa(p.Page))

	if p.Page > 1 {
		prevPage := p.Page - 1
		headers.Add("X-Prev-Page", strconv.Itoa(prevPage))
	}

	if p.Page < totalPages {
		nextPage := p.Page + 1
		headers.Add("X-Next-Page", strconv.Itoa(nextPage))
	}
}

func (k *contextKey) String() string {
	return "go/subs/http context value " + k.name
}

func RandomStr(length int) (string, error) {
	bytes := make([]byte, length)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = chars[b%byte(len(chars))]
	}

	return string(bytes), nil
}
