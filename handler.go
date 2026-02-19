package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	host := "http://localhost:9000"
	w.Header().Set("Content-Type", "text/html")
	log.Println("Get Home")
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)

	db.mu.RLock()
	var b strings.Builder
	for shortID, link := range db.links {
		fmt.Fprintf(
			&b,
			`<p>Original: %s<br>
             Shorten: <a href='%s/shorten/%s'>%s/shorten/%s</a><br></p>`,
			link, host, shortID, host, shortID,
		)
	}
	db.mu.RUnlock()

	fmt.Fprint(w, "<h2>Hello and Welcome to the Go URL Shortener!<h2>\n")
	fmt.Fprint(w, b.String())
}
