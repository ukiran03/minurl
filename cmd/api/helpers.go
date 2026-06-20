package main

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) readSlugParam(r *http.Request) string {
	return httprouter.ParamsFromContext(r.Context()).ByName("slug")
}

type envelope map[string]any

func (app *application) writeJSON(
	w http.ResponseWriter, status int, data envelope, headers http.Header,
) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	js = append(js, '\n')

	for key, values := range headers {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}
