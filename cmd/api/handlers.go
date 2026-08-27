package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"ukiran.com/minurl/internal/data"
	"ukiran.com/minurl/internal/flake"
	"ukiran.com/minurl/internal/stream"
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

	urlHash := getURLHash(inURL)

	// Query the Bloom Filter (Fail-open if error occurs)
	exists, err := app.bloom.Exists(r.Context(), urlHash)
	if err != nil {
		app.logger.Error("bloom filter check failed", "error", err)
	}

	if exists {
		// Verify against Redis using our secondary index
		if storedMinUrl, err := app.models.Cache.GetByHash(
			r.Context(),
			urlHash,
		); err == nil &&
			storedMinUrl != nil {

			fullShortURL := fmt.Sprintf(
				"%s/%s",
				app.config.BaseURL,
				storedMinUrl.Slug,
			)

			app.writeJSON(
				w, http.StatusOK,
				envelope{
					"url":       storedMinUrl.URL,
					"short_url": fullShortURL,
				}, nil,
			)
			return
		}

		// Redis miss (eviction), check Postgres. In theory only 1% of the
		// requests will reach this end, 99% of them will be caught by the
		// above redis check.
		dbFlake, longURL, expiry, err := app.models.PostgresDB.GetByHashWithAll(
			r.Context(),
			urlHash,
		)
		if err == nil {
			// convert raw Snowflake int64 to original Base62 string
			originalSlug := flake.Flake(dbFlake).Base62()

			// rehydrate the MinUrl struct to repopulate redis cache
			var expiryVal time.Time
			if expiry != nil {
				expiryVal = *expiry
			}
			lifespan := data.Lifespan{Expiry: expiryVal}
			rehydratedMinUrl := &data.MinUrl{
				Flake:   dbFlake,
				Slug:    originalSlug,
				URL:     longURL,
				URLHash: urlHash,
				Life:    lifespan,
			}

			// Heal the cache asynchronously or synchronously so future reads
			// hit redis
			if cacheErr := app.models.Cache.Put(
				r.Context(),
				rehydratedMinUrl,
			); cacheErr != nil {
				app.logger.Error(
					"failed to heal cache on eviction fallback",
					"error",
					cacheErr,
				)
			}

			fullShortURL := fmt.Sprintf(
				"%s/%s",
				app.config.BaseURL,
				originalSlug,
			)

			app.writeJSON(w, http.StatusOK, envelope{
				"url":       longURL,
				"short_url": fullShortURL,
			}, nil)
			return
		}

		// NOTE: If Postgres returned ErrRecordNotFound, it's a true Bloom
		// filter false positive. Proceed below to mint a brand new record
		// safely!
		//
		// For, How we are dealing with unique constraints violations,
		// see `PostgresStore.Copy()`
	}

	// Core Cache Miss -> Instantly generate our unique SFID & Slug
	minurl := data.NewMinUrl(int64(app.sfid), inURL, lifespan)
	minurl.URLHash = urlHash

	payload, err := json.Marshal(minurl)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Update Cache layers FIRST, so immediate reads won't hit a DB race
	if err := app.models.Cache.Put(r.Context(), minurl); err != nil {
		app.logger.Error("failed to update redis cache", "error", err)
		app.serverErrorResponse(
			w,
			r,
			errors.New("failed to process short URL cache"),
		)
		return
	}

	// Publish to Jetstream for DB async ingestion AFTER cache is secured
	err = app.postgresStream.Publish(
		r.Context(),
		stream.PgsSubjectName,
		payload,
	)
	if err != nil {
		app.logger.Error("postgres stream publish failed", "error", err)
		// TODO: Consider rolling back or handling stale cache if publishing fails
		app.serverErrorResponse(
			w,
			r,
			errors.New("failed to queue postgres database write"),
		)
		return
	}

	if err := app.bloom.Add(r.Context(), urlHash); err != nil {
		app.logger.Error("failed to add to bloom filter", "error", err)
	}

	fullShortURL := fmt.Sprintf("%s/%s", app.config.BaseURL, minurl.Slug)

	// Return 201 Created using a helper if available, or standard encoding
	app.writeJSON(w, http.StatusCreated, envelope{
		"url":       minurl.URL,
		"short_url": fullShortURL,
	}, nil)
}

// GET /{slug}
func (app *application) redirectHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		app.notFoundResponse(w, r)
		return
	}

	// Hit Redis Cache first
	if cachedMinUrl, err := app.models.Cache.GetBySlug(
		r.Context(),
		slug,
	); err == nil &&
		cachedMinUrl != nil {
		app.executeRedirect(w, r, slug, cachedMinUrl.URL)
		return
	}

	// Fall through to Postgres DB
	f, err := flake.ParseBase62(slug)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	longUrl, err := app.models.PostgresDB.Get(
		r.Context(),
		&data.MinUrl{Flake: int64(f)},
	)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	app.executeRedirect(w, r, slug, longUrl)
}

// Helper to keep redirection and analytics publishing DRY
func (app *application) executeRedirect(
	w http.ResponseWriter, r *http.Request, slug, targetURL string,
) {
	clickEvent := data.NewClickEvent(slug, r)

	payload, err := json.Marshal(clickEvent)
	if err != nil {
		app.logger.Error("failed to marshal click event", "error", err)
	} else {
		// using a detached background context so the publish isn't aborted
		// when the HTTP request context finishes upon redirection.
		bgCtx := context.WithoutCancel(r.Context())

		if err := app.clickhouseStream.Publish(
			bgCtx, stream.ChStreamSubjectName, payload,
		); err != nil {
			app.logger.Error("clickhouse stream publish failed", "error", err)
		}
	}

	// perform redirect (StatusFound / 302 used here to force browser re-visits
	// for metrics tracking)
	http.Redirect(w, r, targetURL, http.StatusFound)
}

// These handlers below will be used to retrive information/stats of minurls

// GET /v1/minurls/{slug}
// Get analytics for a specific slug
func (app *application) getMinurlHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input GetDTO
	input.Slug = chi.URLParam(r, "slug")

	if err := app.readQuery(r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if input.Validate(v); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	stats, err := app.models.ClickhouseDB.GetClickStats(
		r.Context(), input.Slug, input.From, input.To, input.Limit,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, envelope{
		"total_clicks":  stats.TotalClicks,
		"top_referrers": stats.TopReferrers,
	}, nil)
}

// DELETE /v1/minurls/{slug}
func (app *application) deleteMinurlHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	slug := chi.URLParam(r, "slug")
	m := &data.MinUrl{Slug: slug}

	// Delete from Redis cache
	if err := app.models.Cache.Delete(r.Context(), m); err != nil {
		app.logger.Error("failed to delete from cache", "error", err)
	}

	// Synchronous delete or mark inactive in PostgreSQL
	if err := app.models.PostgresDB.Delete(r.Context(), m); err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	app.writeJSON(
		w,
		http.StatusOK,
		envelope{"message": "minurl deleted successfully"},
		nil,
	)
}

// [25-08-2026] TODO: In the envelope with "short_url", short_url is only the
// slug, not a full URL, change it.
