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
		r.Post("/shorten", app.createMinurlHandler) // -> POST /v1/shorten

		// SUB-BRANCH 3: Protected URL Management
		r.Route("/minurls/{slug}", func(r chi.Router) {
			r.Get("/", app.getMinurlHandler)       // -> GET /v1/minurls/{slug}
			r.Delete("/", app.deleteMinurlHandler) // -> DELETE /v1/minurls/{slug}
		})
	})

	return r
}
