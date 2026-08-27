package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(app.recoverPanic)

	r.NotFound(app.notFoundResponse)
	r.MethodNotAllowed(app.methodNotAllowedResponse)

	// BRANCH 1: Global public redirect (No heavy middleware)
	r.Get("/{slug}", app.redirectHandler)

	// BRANCH 2: The API Group
	r.Route("/v1", func(r chi.Router) {
		r.Get("/healthcheck", app.healthcheckHandler)
		r.Post("/shorten", app.createMinurlHandler)

		// Explicit full paths prevent trailing slash bugs
		r.Get("/minurls/{slug}", app.getMinurlHandler)
		r.Delete("/minurls/{slug}", app.deleteMinurlHandler)
	})

	return r
}
