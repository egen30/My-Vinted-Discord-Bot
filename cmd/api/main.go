package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/2spy/vinted-discord-bot/internal/store"
	"github.com/2spy/vinted-discord-bot/pkg/models"
)

func main() {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		panic("DATABASE_URL is required")
	}
	ctx := context.Background()
	db, err := store.NewPostgresStore(ctx, databaseURL)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	token := strings.TrimSpace(os.Getenv("API_TOKEN"))
	handler := &searchHandler{store: db, token: token}
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		panic(err)
	}
}

type searchHandler struct {
	store *store.PostgresStore
	token string
}

func (h *searchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.token != "" && r.Header.Get("Authorization") != "Bearer "+h.token {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/searches":
		h.list(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/searches":
		h.create(w, r)
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/searches/") && strings.HasSuffix(r.URL.Path, "/enabled"):
		h.setEnabled(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/searches/"):
		h.delete(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *searchHandler) list(w http.ResponseWriter, r *http.Request) {
	searches, err := h.store.ListSearches(r.Context(), r.URL.Query().Get("enabled") == "true")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, searches)
}

func (h *searchHandler) create(w http.ResponseWriter, r *http.Request) {
	var input models.Search
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := validateSearch(input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created, err := h.store.CreateSearch(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *searchHandler) setEnabled(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/searches/"), "/enabled"), 10, 64)
	if err != nil {
		http.Error(w, "invalid search ID", http.StatusBadRequest)
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.store.SetSearchEnabled(r.Context(), id, body.Enabled); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *searchHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/searches/"), 10, 64)
	if err != nil {
		http.Error(w, "invalid search ID", http.StatusBadRequest)
		return
	}
	if err := h.store.DeleteSearch(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateSearch(search models.Search) error {
	if strings.TrimSpace(search.Name) == "" {
		return fmt.Errorf("name is required")
	}
	parsed, err := url.Parse(strings.TrimSpace(search.URL))
	if err != nil || parsed.Scheme != "https" || !strings.HasSuffix(parsed.Hostname(), ".vinted.de") && parsed.Hostname() != "vinted.de" {
		return fmt.Errorf("url must be an HTTPS Vinted domain")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
