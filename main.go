package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
)

// Use a struct to group the map and a mutex for thread safety
type Map struct {
	mu    sync.Mutex
	links map[string]string
}

var db = Map{
	links: make(map[string]string),
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/addLink", addLinkHandler)
	mux.HandleFunc("/shorten/", getLinkHandler)

	log.Fatal(http.ListenAndServe(":9000", mux))
}

func addLinkHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("link")
	if url == "" {
		http.Error(w, "Missing 'link' query parameter", http.StatusBadRequest)
		return
	}

	db.mu.Lock()
	// Generate a simple 4-character hex string for the toy example
	shortID := fmt.Sprintf("%x", rand.Intn(0xFFFF))
	db.links[shortID] = url
	db.mu.Unlock()

	shortURL := fmt.Sprintf("http://localhost:9000/shorten/%s", shortID)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `Added! Short link: <a href="%s">%s</a>`, shortURL, shortURL)
}

func getLinkHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the ID by trimming the prefix
	id := strings.TrimPrefix(r.URL.Path, "/shorten/")

	db.mu.Lock()
	originalURL, exists := db.links[id]
	db.mu.Unlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}
