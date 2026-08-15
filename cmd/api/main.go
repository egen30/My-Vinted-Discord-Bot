package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
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
	case r.Method == http.MethodGet && (r.URL.Path == "/" || r.URL.Path == "/admin"):
		h.adminPage(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/searches":
		h.list(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/searches":
		h.create(w, r)
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/searches/"):
		h.update(w, r)
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/searches/") && strings.HasSuffix(r.URL.Path, "/enabled"):
		h.setEnabled(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/searches/"):
		h.delete(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/admin/searches":
		h.createFromForm(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/admin/searches/"):
		h.adminAction(w, r)
	default:
		http.NotFound(w, r)
	}
}

type adminPageData struct {
	Searches []models.Search
	Listings []models.ListingSummary
	Error    string
}

func (h *searchHandler) adminPage(w http.ResponseWriter, r *http.Request) {
	searches, err := h.store.ListSearches(r.Context(), false)
	if err != nil { http.Error(w, "could not load searches", http.StatusInternalServerError); return }
	listings, err := h.store.RecentListings(r.Context(), 50)
	if err != nil { http.Error(w, "could not load listings", http.StatusInternalServerError); return }
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := adminTemplate.Execute(w, adminPageData{Searches: searches, Listings: listings}); err != nil {
		// The response may already have started; logging is intentionally omitted
		// because the page contains no sensitive request data.
		return
	}
}

func (h *searchHandler) createFromForm(w http.ResponseWriter, r *http.Request) {
	priority, _ := strconv.Atoi(r.FormValue("priority"))
	input := models.Search{Name: strings.TrimSpace(r.FormValue("name")), URL: strings.TrimSpace(r.FormValue("url")), Notes: strings.TrimSpace(r.FormValue("notes")), Priority: priority, Enabled: r.FormValue("enabled") == "on"}
	if err := validateSearch(input); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	if _, err := h.store.CreateSearch(r.Context(), input); err != nil { http.Error(w, err.Error(), http.StatusConflict); return }
	h.redirectAdmin(w, r)
}

func (h *searchHandler) adminAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/admin/searches/"), "/")
	if len(parts) != 2 { http.NotFound(w, r); return }
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil { http.Error(w, "invalid search ID", http.StatusBadRequest); return }
	switch parts[1] {
	case "toggle":
		searches, listErr := h.store.ListSearches(r.Context(), false)
		if listErr != nil { http.Error(w, listErr.Error(), http.StatusInternalServerError); return }
		for _, search := range searches { if search.ID == id { err = h.store.SetSearchEnabled(r.Context(), id, !search.Enabled); break } }
	case "delete":
		err = h.store.DeleteSearch(r.Context(), id)
	default:
		http.NotFound(w, r); return
	}
	if err != nil { http.Error(w, err.Error(), http.StatusNotFound); return }
	h.redirectAdmin(w, r)
}

func (h *searchHandler) redirectAdmin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
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

func (h *searchHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/searches/"), 10, 64)
	if err != nil {
		http.Error(w, "invalid search ID", http.StatusBadRequest)
		return
	}
	var input models.Search
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := validateSearch(input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	input.ID = id
	updated, err := h.store.UpdateSearch(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, updated)
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

var adminTemplate = template.Must(template.New("admin").Funcs(template.FuncMap{"div": func(cents int64, divisor int64) float64 { return float64(cents) / float64(divisor) }}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Vinted Monitor Admin</title>
<style>body{font:16px system-ui,sans-serif;max-width:1100px;margin:2rem auto;padding:0 1rem;color:#18202a}section{margin:2rem 0}table{border-collapse:collapse;width:100%}th,td{padding:.6rem;border-bottom:1px solid #ddd;text-align:left;vertical-align:top}input{padding:.5rem;margin:.2rem;width:min(100%,30rem)}button{padding:.45rem .7rem}.ok{color:#087f5b}.bad{color:#c92a2a}.muted{color:#68717d;font-size:.9rem}</style></head>
<body><h1>Vinted listing monitor</h1>
<section><h2>Sourcing searches</h2><form method="post" action="/admin/searches"><input name="name" placeholder="Search name" required><input name="url" type="url" placeholder="https://www.vinted.de/catalog?..." required><input name="priority" type="number" value="0" title="Priority"><input name="notes" placeholder="Notes"><label><input name="enabled" type="checkbox" checked> enabled</label><button>Add search</button></form>
<table><tr><th>Status</th><th>Name</th><th>URL</th><th>Health</th><th>Actions</th></tr>{{range .Searches}}<tr><td>{{if .Enabled}}<span class="ok">Enabled</span>{{else}}<span class="muted">Disabled</span>{{end}}</td><td>{{.Name}}<br><span class="muted">priority {{.Priority}}</span></td><td><a href="{{.URL}}">{{.URL}}</a></td><td>{{if .LastError}}<span class="bad">{{.LastError}}</span>{{else if .LastSuccessfulAt}}last successful {{.LastSuccessfulAt}}{{else}}<span class="muted">not checked</span>{{end}}</td><td><form method="post" action="/admin/searches/{{.ID}}/toggle" style="display:inline"><button>{{if .Enabled}}Disable{{else}}Enable{{end}}</button></form> <form method="post" action="/admin/searches/{{.ID}}/delete" style="display:inline"><button>Delete</button></form></td></tr>{{else}}<tr><td colspan="5">No searches configured.</td></tr>{{end}}</table></section>
<section><h2>Recent listings</h2><table><tr><th>First seen</th><th>Listing</th><th>Price</th><th>Found through</th></tr>{{range .Listings}}<tr><td>{{.FirstSeenAt}}</td><td><a href="{{.URL}}">{{.Title}}</a><br><span class="muted">{{.Seller}} · {{.ExternalID}}</span></td><td>{{printf "%.2f" (div .PriceCents 100)}} {{.Currency}}</td><td>{{range .SearchNames}}{{.}}<br>{{end}}</td></tr>{{else}}<tr><td colspan="4">No listings recorded yet.</td></tr>{{end}}</table></section>
<section><h2>Optional modules</h2><p class="muted">Historical data, profitability, image analysis, and Deal Score are optional and do not gate discovery or notifications.</p></section>
</body></html>`))
