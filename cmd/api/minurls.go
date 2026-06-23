package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"ukiran.com/minurl/internal/data"
)

// POST /v1/shorten
func (app *application) createMinurlHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	fmt.Fprintf(w, "create a new minurl for %s", slug)
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
	fmt.Fprintf(w, "get the URL for %s", slug)
}

// DELETE /v1/minurls/{slug}
func (app *application) deleteMinurlHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	fmt.Fprintf(w, "delete the minURL %s", slug)
	return
}
