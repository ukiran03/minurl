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
	var minurl data.MinUrl

	err := app.readJSON(w, r, &minurl)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// At this exact point, minurl.Created is set to time.Now()
	// and minurl.Expiry has been calculated and jittered!
	// see data.Minurl.UnmarshalJSON()

	v := validator.New()

	// URL validation
	v.Check(minurl.URL != "", "url", "must be provided")
	v.Check(
		len(minurl.URL) >= 11 && len(minurl.URL) <= 2048,
		"url", "must be between 11 and 2048 characters long",
	)

	// Slug validation
	if minurl.IsCustom {
		v.Check(len(minurl.Slug) >= 8 && len(minurl.Slug) <= 100,
			"custom slug", "must be between 8 and 100 characters long")
		v.Check(!validator.Matches(minurl.Slug, validator.SlugRegex),
			"custom slug", ErrInvalidCustomSlug.Error())
	}

	// Title validation (Using pointer safe-check)
	if minurl.Title != nil && *minurl.Title != "" {
		v.Check(len(*minurl.Title) <= 100,
			"title", "must not be more than 100 characters long")
	}

	// UserID validation
	if minurl.UserID != nil {
		v.Check(*minurl.UserID > 0, "user_id", "must be a valid positive integer")
	}

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
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

		UserID: new(int64(1)),
		Lifespan: data.Lifespan{
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
