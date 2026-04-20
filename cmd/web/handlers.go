package main

import (
	"fmt"
	"net/http"
	"strconv"

	"ukiran.com/minurl/ui/html/pages"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	murls, err := app.murls.Latest(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	w.Header().Add("Server", "Go")

	// var b bytes.Buffer
	// for _, murl := range murls {
	// 	b.Write([]byte(murl.String()))
	// }
	// b.WriteTo(w)

	err = pages.HomePage(
		"Home", "There's nothing to see here yet!", murls).Render(r.Context(), w)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
}

func (app *application) minurlView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	msg := fmt.Sprintf("Display a specific MinUrl with ID %d...", id)
	w.Write([]byte(msg))
}

func (app *application) minurlCreate(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Display a form for creating a new MinUrl..."))
}

func (app *application) minurlCreatePost(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Save a new MinUrl..."))
}
