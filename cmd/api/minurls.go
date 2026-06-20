package main

import (
	"fmt"
	"net/http"
)

// POST /v1/minurls
func (app *application) createMinurlHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "create a new minurl")
}

// GET /r/:slug
func (app *application) redirectHandler(w http.ResponseWriter, r *http.Request) {
	slug := app.readSlugParam(r)

	fmt.Fprintf(w, "slug is %s\n", slug)
}
