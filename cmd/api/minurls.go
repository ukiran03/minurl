package main

import (
	"fmt"
	"net/http"

	"ukiran.com/minurl/internal/data"
)

// POST /v1/shorten
func (app *application) createMinurlHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "create a new minurl")
}

// GET /{slug}
func (app *application) redirectHandler(w http.ResponseWriter, r *http.Request) {
	slug := app.readSlugParam(r)

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
	return
}

// DELETE /v1/minurls/{slug}
func (app *application) deleteMinurlHandler(w http.ResponseWriter, r *http.Request) {
	return
}
