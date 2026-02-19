package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Use a struct to group the map and a mutex for thread safety
type Map struct {
	mu    sync.RWMutex
	links map[string]string
}

var db = Map{
	links: make(map[string]string),
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/addLink", addLinkHandler)
	mux.HandleFunc("/shorten/", getLinkHandler)
	mux.HandleFunc("/", homeHandler)
	log.Fatal(http.ListenAndServe(":9000", mux))
}

func addLinkHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("link")
	if url == "" {
		http.Error(w, "Missing 'link' query parameter", http.StatusBadRequest)
		return
	}
	db.mu.Lock()
	shortID := randomID()
	db.links[shortID] = url
	db.mu.Unlock()

	shortURL := fmt.Sprintf("http://localhost:9000/shorten/%s", shortID)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `Added! Short link: <a href="%s">%s</a>`, shortURL, shortURL)
}

func getLinkHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the ID by trimming the prefix
	id := strings.TrimPrefix(r.URL.Path, "/shorten/")

	db.mu.RLock()
	originalURL, exists := db.links[id]
	db.mu.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func randomID() string {
	b := make([]byte, 4) // 4 bytes = 8 hex characters
	if _, err := rand.Read(b); err != nil {
		return "00000000" // TODO: handle error
	}
	return hex.EncodeToString(b)
}
