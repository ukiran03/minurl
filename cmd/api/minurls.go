package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"ukiran.com/minurl/internal/data"
	"ukiran.com/minurl/internal/validator"
)

// POST /v1/shorten
func (app *application) createMinurlHandler(w http.ResponseWriter, r *http.Request) {
	var input minurlRequestDTO
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	input.Validate(v)

	lifespan, err := data.NewLifespan(input.Expiry)
	if err != nil {
		v.AddError("expires_at", err.Error())
	}
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Map the DTO directly into your Clean Domain Struct
	minurl := data.MinUrl{
		Slug:     input.Slug,
		URL:      input.URL,
		Title:    input.Title,
		IsCustom: input.IsCustom,
		UserID:   input.UserID,
		Life:     lifespan,
	}

	// Now you can pass `minurl` directly to your database insert function!
	fmt.Fprintf(w, "%+v\n", minurl)
}

// GET /{slug}
func (app *application) redirectHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	fmt.Fprintf(w, "%s", slug)
}

// GET /v1/minurls/{slug}
func (app *application) getMinurlHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	minurl := data.MinUrl{
		Slug:     slug,
		URL:      "https://example.com",
		Title:    new("Example"),
		IsCustom: false,
		UserID:   new(int64(1)),

		Life: data.Lifespan{
			Created: time.Now(),
			Expiry:  time.Now().AddDate(0, 0, 3),
		},
	}

	err := app.writeJSON(w, http.StatusOK, envelope{"minurl": minurl}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// DELETE /v1/minurls/{slug}
func (app *application) deleteMinurlHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	fmt.Fprintf(w, "delete the minURL %s", slug)
}
