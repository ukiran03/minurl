package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"ukiran.com/minurl/internal/data"
	"ukiran.com/minurl/internal/validator"
)

// POST /v1/shorten
func (app *application) createMinurlHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input requestDTO
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if input.Validate(v); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	lifespan, err := data.NewLifespan(input.Expiry)
	if err != nil {
		v.AddError("expires_at", err.Error())
	}

	inURL, err := processURL(input.URL)
	if err != nil {
		v.AddError("invalid_url", err.Error())
	}
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Generate the URLhash early to perform lookups
	urlHash := getURLHash(inURL)

	// Query the Bloom Filter
	exists, err := app.bloom.Exists(r.Context(), urlHash)
	if err != nil {
		app.logger.Error("bloom filter check failed", "error", err)
	}

	if exists {
		// Verify against Redis using our secondary index
		storedMinUrl, err := app.models.Cache.GetByHash(r.Context(), urlHash)
		if err == nil && storedMinUrl != nil {
			// Idempotent Match Found: Return 200 OK with the existing resource data
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(storedMinUrl); err != nil {
				app.logger.Error("failed to encode response", "error", err)
			}
			return
		}
		// If it was a false positive, we fall through safely
	}

	// Core Cache Miss -> Instantly generate our unique SFID & Slug
	minurl := data.NewMinUrl(int64(app.sfid), inURL, lifespan)
	minurl.URLHash = urlHash // Assign the pre-computed hash

	payload, err := json.Marshal(minurl)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Publish to Jetstream for DB asynv ingestion
	err = app.stream.Publish(r.Context(), "minurl.created", payload)
	if err != nil {
		app.logger.Error("jetstream publish failed", "error", err)
		app.serverErrorResponse(
			w, r, errors.New("failed to queue database write"),
		)
		return
	}
	/* app.logger.Info("message persisted in stream",
	   "stream", pubAck.Stream, "seq", pubAck.Sequence) */

	// Update Cache layers (Populates both the Hash structure and index)
	if err := app.models.Cache.Put(r.Context(), minurl); err != nil {
		app.logger.Error("failed to update redis cache", "error", err)
	}

	if err := app.bloom.Add(r.Context(), urlHash); err != nil {
		app.logger.Error("failed to add to bloom filter", "error", err)
	}

	// Return 201 Created
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(minurl); err != nil {
		app.logger.Error("failed to encode response", "error", err)
	}
}

// GET /{slug}
func (app *application) redirectHandler(
	w http.ResponseWriter,
	r *http.Request,
)

// GET /v1/minurls/{slug}
func (app *application) getMinurlHandler(
	w http.ResponseWriter,
	r *http.Request,
)

// DELETE /v1/minurls/{slug}
func (app *application) deleteMinurlHandler(
	w http.ResponseWriter,
	r *http.Request,
)
