package main

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					panic(rvr)
				}

				app.logger.Error(
					"panic recovered",
					"panic", rvr,
					"stack", string(debug.Stack()),
				)

				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%v", rvr))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
