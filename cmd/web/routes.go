package main

import (
	"net/http"

	"ukiran.com/minurl/ui"
)

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(ui.Files)))
	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /minurl/view/{id}", app.minurlView)
	mux.HandleFunc("GET /minurl/create", app.minurlCreate)
	mux.HandleFunc("POST /minurl/create", app.minurlCreatePost)
	return mux
}
