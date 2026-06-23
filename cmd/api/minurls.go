package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"ukiran.com/minurl/internal/data"
)

// POST /v1/shorten
func (app *application) createMinurlHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Slug      string     `json:"slug"`
		URL       string     `json:"url"`
		Title     *string    `json:"title,omitzero"`
		IsCustom  bool       `json:"is_custom"`
		ExpiresAt *time.Time `json:"expires_at"`
		UserID    *int64     `json:"user_id,omitzero"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	fmt.Fprintf(w, "%+v\n", input)
}

// GET /{slug}
func (app *application) redirectHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	longURL, exists := data.UrlDatabase[slug]
	if !exists {
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	}

	// Optional: Track analytics here asynchronously
	// go trackClick(shortCode, r)

	// Redirect the user using a 302 Found status
	http.Redirect(w, r, longURL, http.StatusFound)
}

// GET /v1/minurls/{slug}
func (app *application) getMinurlHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	minurl := data.MinUrl{
		Slug:      slug,
		URL:       "https://example.com",
		Title:     new("Example"),
		IsCustom:  false,
		CreatedAt: time.Now(),
		ExpiresAt: new(time.Now().AddDate(0, 0, 3)),
		UserID:    new(int64(1)),
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
