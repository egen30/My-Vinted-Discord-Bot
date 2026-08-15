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
	case r.Method == http.MethodPost && r.URL.Path == "/admin/history":
		h.saveHistoryFromForm(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/admin/searches/"):
		h.adminAction(w, r)
	default:
		http.NotFound(w, r)
	}
}

type adminPageData struct {
	Searches []models.Search
	History  models.HistorySource
	Error    string
}

func (h *searchHandler) adminPage(w http.ResponseWriter, r *http.Request) {
	searches, err := h.store.ListSearches(r.Context(), false)
	if err != nil {
		http.Error(w, "could not load searches", http.StatusInternalServerError)
		return
	}
	historySource, err := h.store.GetHistorySource(r.Context())
	if err != nil {
		http.Error(w, "could not load history settings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := adminTemplate.Execute(w, adminPageData{Searches: searches, History: historySource}); err != nil {
		// The response may already have started; logging is intentionally omitted
		// because the page contains no sensitive request data.
		return
	}
}

func (h *searchHandler) saveHistoryFromForm(w http.ResponseWriter, r *http.Request) {
	spreadsheetURL := strings.TrimSpace(r.FormValue("spreadsheet_url"))
	worksheet := strings.TrimSpace(r.FormValue("worksheet"))
	if err := validateHistorySource(spreadsheetURL, worksheet); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := h.store.SaveHistorySource(r.Context(), models.HistorySource{SpreadsheetURL: spreadsheetURL, Worksheet: worksheet, Enabled: r.FormValue("enabled") == "on"}); err != nil {
		http.Error(w, "could not save history settings", http.StatusInternalServerError)
		return
	}
	h.redirectAdmin(w, r)
}

func (h *searchHandler) createFromForm(w http.ResponseWriter, r *http.Request) {
	priority, _ := strconv.Atoi(r.FormValue("priority"))
	input := models.Search{Name: strings.TrimSpace(r.FormValue("name")), URL: strings.TrimSpace(r.FormValue("url")), Notes: strings.TrimSpace(r.FormValue("notes")), Priority: priority, Enabled: r.FormValue("enabled") == "on"}
	if err := validateSearch(input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := h.store.CreateSearch(r.Context(), input); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	h.redirectAdmin(w, r)
}

func (h *searchHandler) adminAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/admin/searches/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid search ID", http.StatusBadRequest)
		return
	}
	switch parts[1] {
	case "toggle":
		searches, listErr := h.store.ListSearches(r.Context(), false)
		if listErr != nil {
			http.Error(w, listErr.Error(), http.StatusInternalServerError)
			return
		}
		found := false
		for _, search := range searches {
			if search.ID == id {
				err = h.store.SetSearchEnabled(r.Context(), id, !search.Enabled)
				found = true
				break
			}
		}
		if !found {
			err = fmt.Errorf("search %d not found", id)
		}
	case "update":
		priority, _ := strconv.Atoi(r.FormValue("priority"))
		input := models.Search{ID: id, Name: strings.TrimSpace(r.FormValue("name")), URL: strings.TrimSpace(r.FormValue("url")), Notes: strings.TrimSpace(r.FormValue("notes")), Priority: priority, Enabled: r.FormValue("enabled") == "on"}
		if validationErr := validateSearch(input); validationErr != nil {
			http.Error(w, validationErr.Error(), http.StatusBadRequest)
			return
		}
		_, err = h.store.UpdateSearch(r.Context(), input)
	case "delete":
		err = h.store.DeleteSearch(r.Context(), id)
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
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

func validateHistorySource(spreadsheetURL, worksheet string) error {
	parsed, err := url.Parse(strings.TrimSpace(spreadsheetURL))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "docs.google.com" || !strings.HasPrefix(parsed.Path, "/spreadsheets/d/") {
		return fmt.Errorf("spreadsheet URL must be an HTTPS Google Sheets URL")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "spreadsheets" || parts[1] != "d" || strings.TrimSpace(parts[2]) == "" {
		return fmt.Errorf("spreadsheet URL must include a spreadsheet ID")
	}
	if strings.TrimSpace(worksheet) == "" {
		return fmt.Errorf("worksheet is required")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

var adminTemplate = template.Must(template.New("admin").Funcs(template.FuncMap{
	"activeCount": func(searches []models.Search) int {
		count := 0
		for _, search := range searches {
			if search.Enabled {
				count++
			}
		}
		return count
	},
}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Vinted Scout · Admin</title>
<style>
:root{color-scheme:dark;--bg:#07111f;--panel:#101d31;--panel2:#142641;--line:#263953;--text:#edf4ff;--muted:#91a3bc;--teal:#63e6d1;--blue:#7aa8ff;--red:#ff8f9c;--shadow:0 18px 50px #02071180}
*{box-sizing:border-box}body{margin:0;min-height:100vh;background:radial-gradient(circle at 90% 0,#183961 0,transparent 35%),linear-gradient(145deg,var(--bg),#0b1729 65%,#0e2039);font:15px/1.5 Inter,ui-sans-serif,system-ui,sans-serif;color:var(--text)}a{color:var(--teal);text-decoration:none}a:hover{text-decoration:underline}.shell{max-width:1240px;margin:0 auto;padding:34px 22px 60px}.topbar{display:flex;align-items:flex-end;justify-content:space-between;gap:20px;margin-bottom:28px}.eyebrow{margin:0 0 6px;color:var(--teal);font-size:12px;font-weight:800;letter-spacing:.16em;text-transform:uppercase}.topbar h1{margin:0;font-size:clamp(30px,5vw,48px);letter-spacing:-.04em}.lede{margin:8px 0 0;color:var(--muted);max-width:620px}.pulse{display:flex;align-items:center;gap:9px;color:var(--muted);font-size:13px}.dot{width:9px;height:9px;border-radius:50%;background:var(--teal);box-shadow:0 0 0 5px #63e6d11c}.stats{display:grid;grid-template-columns:repeat(2,1fr);gap:14px;margin-bottom:22px}.stat,.panel{background:linear-gradient(145deg,#142640e8,#0f1c2fe8);border:1px solid var(--line);box-shadow:var(--shadow);border-radius:18px}.stat{padding:18px 20px}.stat-label{color:var(--muted);font-size:12px;text-transform:uppercase;letter-spacing:.08em}.stat-value{display:block;margin-top:3px;font-size:29px;font-weight:750}.layout{display:grid;grid-template-columns:minmax(0,1.4fr) minmax(310px,.8fr);gap:18px;align-items:start}.panel{padding:22px}.panel h2{margin:0;font-size:19px;letter-spacing:-.02em}.panel-head{display:flex;justify-content:space-between;align-items:center;gap:12px;margin-bottom:17px}.badge{padding:4px 9px;border-radius:999px;background:#7aa8ff1c;color:var(--blue);font-size:12px;font-weight:700}.form-grid{display:grid;grid-template-columns:1fr 1.5fr 100px;gap:10px}.field{min-width:0}.field label{display:block;color:var(--muted);font-size:12px;margin-bottom:5px}.field input{width:100%}input{border:1px solid var(--line);background:#0a1627;color:var(--text);border-radius:10px;padding:10px 11px;outline:none}input:focus{border-color:var(--teal);box-shadow:0 0 0 3px #63e6d11c}.full{grid-column:1/-1}.check{display:flex;align-items:center;gap:8px;color:var(--muted);font-size:13px}.check input{width:auto;accent-color:var(--teal)}button{border:0;border-radius:10px;padding:10px 14px;background:linear-gradient(135deg,#5de0ca,#55a7ff);color:#061322;font-weight:800;cursor:pointer}button:hover{filter:brightness(1.1)}button.ghost{background:#1a2d47;color:var(--text);border:1px solid var(--line)}button.danger{background:#3a1d2a;color:var(--red);border:1px solid #6b3044}.primary{margin-top:12px}.search-list{display:grid;gap:10px;margin-top:18px}.search-card{border:1px solid var(--line);background:#0c192b;border-radius:14px;padding:15px;display:grid;grid-template-columns:1fr auto;gap:14px}.search-title{display:flex;align-items:center;gap:9px;font-weight:750}.status{width:8px;height:8px;border-radius:50%;background:var(--muted)}.status.on{background:var(--teal);box-shadow:0 0 0 4px #63e6d11c}.search-url,.meta{color:var(--muted);font-size:12px;overflow-wrap:anywhere}.search-url{margin:4px 0 10px}.actions{display:flex;flex-wrap:wrap;gap:7px;justify-content:flex-end}.actions form{display:inline}.actions button{padding:7px 10px;font-size:12px}.edit{grid-column:1/-1;border-top:1px solid var(--line);padding-top:13px}.edit summary{color:var(--blue);cursor:pointer;font-size:13px}.edit-grid{display:grid;grid-template-columns:1fr 1.5fr 90px;gap:9px;margin-top:12px}.health{font-size:12px;color:var(--muted)}.error{color:var(--red)}.module{display:flex;gap:12px;padding:14px 0;border-bottom:1px solid var(--line)}.module:last-child{border-bottom:0}.module-icon{display:grid;place-items:center;width:34px;height:34px;border-radius:10px;background:#7aa8ff1c;color:var(--blue)}.module p{margin:2px 0 0;color:var(--muted);font-size:13px}.footer{margin-top:18px;color:var(--muted);font-size:12px;text-align:center}@media(max-width:850px){.layout{grid-template-columns:1fr}.topbar{align-items:flex-start;flex-direction:column}.form-grid,.edit-grid{grid-template-columns:1fr}.stats{grid-template-columns:1fr 1fr}.full{grid-column:auto}}@media(max-width:520px){.shell{padding:24px 14px 40px}.stats{grid-template-columns:1fr}.search-card{grid-template-columns:1fr}.actions{justify-content:flex-start}}
</style></head>
<body><main class="shell"><header class="topbar"><div><p class="eyebrow">Vinted Scout · Control room</p><h1>Find the next great pair.</h1><p class="lede">Manage your sourcing searches, inspect recent finds, and keep optional enrichment safely out of the alert path.</p></div><div class="pulse"><span class="dot"></span>Monitor online</div></header>
<section class="stats"><div class="stat"><span class="stat-label">Configured searches</span><span class="stat-value">{{len .Searches}}</span></div><div class="stat"><span class="stat-label">Active searches</span><span class="stat-value">{{activeCount .Searches}}</span><span class="meta">Enable only the sources you want polled</span></div></section>
<div class="layout"><section class="panel"><div class="panel-head"><div><h2>Sourcing searches</h2><span class="meta">Each enabled URL is monitored independently.</span></div><span class="badge">{{len .Searches}} total</span></div><form method="post" action="/admin/searches"><div class="form-grid"><div class="field"><label for="name">Search name</label><input id="name" name="name" placeholder="e.g. Nike Air Max 42" required></div><div class="field"><label for="url">Vinted catalog URL</label><input id="url" name="url" type="url" placeholder="https://www.vinted.de/catalog?..." required></div><div class="field"><label for="priority">Priority</label><input id="priority" name="priority" type="number" value="0"></div><div class="field full"><label for="notes">Notes</label><input id="notes" name="notes" placeholder="Optional sourcing context"></div></div><label class="check"><input name="enabled" type="checkbox" checked> Start monitoring immediately</label><button class="primary">＋ Add search</button></form><div class="search-list">{{range .Searches}}<article class="search-card"><div><div class="search-title"><span class="status {{if .Enabled}}on{{end}}"></span>{{.Name}} <span class="badge">P{{.Priority}}</span></div><div class="search-url">{{.URL}}</div>{{if .Notes}}<div class="meta">{{.Notes}}</div>{{end}}<div class="health">{{if .LastError}}<span class="error">Issue: {{.LastError}}</span>{{else if .LastSuccessfulAt}}Last successful check: {{.LastSuccessfulAt}}{{else}}Waiting for first check{{end}}</div></div><div class="actions"><form method="post" action="/admin/searches/{{.ID}}/toggle"><button class="ghost">{{if .Enabled}}Pause{{else}}Enable{{end}}</button></form><form method="post" action="/admin/searches/{{.ID}}/delete"><button class="danger">Delete</button></form></div><details class="edit"><summary>Edit search</summary><form method="post" action="/admin/searches/{{.ID}}/update"><div class="edit-grid"><input name="name" value="{{.Name}}" required><input name="url" value="{{.URL}}" type="url" required><input name="priority" type="number" value="{{.Priority}}"><input class="full" name="notes" value="{{.Notes}}"></div><label class="check"><input name="enabled" type="checkbox" {{if .Enabled}}checked{{end}}> Enabled</label><button class="primary">Save changes</button></form></details></article>{{else}}<div class="empty">No searches yet. Add your first Vinted catalog URL above.</div>{{end}}</div></section>
<aside class="panel"><div class="panel-head"><div><h2>Optional modules</h2><span class="meta">Keep enrichment separate from core alerts.</span></div><span class="badge">Optional</span></div><div class="module"><div class="module-icon">↗</div><div><strong>Google Sheets history</strong><p>Configure the optional read-only sales source. Credentials stay in the worker environment.</p><form method="post" action="/admin/history" style="margin-top:10px"><div class="field"><label for="spreadsheet_url">Spreadsheet URL</label><input id="spreadsheet_url" name="spreadsheet_url" type="url" value="{{.History.SpreadsheetURL}}" placeholder="https://docs.google.com/spreadsheets/d/..." required></div><div class="field" style="margin-top:8px"><label for="worksheet">Worksheet / tab</label><input id="worksheet" name="worksheet" value="{{.History.Worksheet}}" placeholder="Sales" required></div><label class="check" style="margin-top:8px"><input name="enabled" type="checkbox" {{if .History.Enabled}}checked{{end}}> Enable history sync</label><button class="primary">Save history settings</button></form>{{if .History.LastSyncAt}}<div class="health">Last sync: {{.History.LastSyncAt}} · {{.History.AcceptedRows}} accepted · {{.History.RejectedRows}} rejected</div>{{end}}{{if .History.LastError}}<div class="health error">Issue: {{.History.LastError}}</div>{{end}}</div></div><div class="module"><div class="module-icon">✓</div><div><strong>Safe by default</strong><p>Alerts never require a model, size, resale estimate, or AI result.</p></div></div></aside></div><p class="footer">Vinted Scout · Configure secrets and database settings in your environment, not in source control.</p></main></body></html>`))
