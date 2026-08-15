// Package historysync synchronizes completed-sales data into an in-memory
// snapshot. The evaluator reads this snapshot and never calls Google Sheets.
package historysync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/2spy/vinted-discord-bot/pkg/history"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type RowsSource interface {
	ReadRows(context.Context) ([][]string, error)
}

type GoogleSheetsSource struct {
	service       *sheets.Service
	spreadsheetID string
	rangeName     string
}

func NewGoogleSheetsSource(ctx context.Context, credentialsJSON []byte, spreadsheetID, worksheet string) (*GoogleSheetsSource, error) {
	if len(credentialsJSON) == 0 || strings.TrimSpace(spreadsheetID) == "" || strings.TrimSpace(worksheet) == "" {
		return nil, fmt.Errorf("Google Sheets credentials, spreadsheet ID, and worksheet are required")
	}
	credentials, err := google.CredentialsFromJSON(ctx, credentialsJSON, sheets.SpreadsheetsReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("parse Google credentials: %w", err)
	}
	service, err := sheets.NewService(ctx, option.WithCredentials(credentials))
	if err != nil {
		return nil, fmt.Errorf("create Google Sheets service: %w", err)
	}
	return &GoogleSheetsSource{service: service, spreadsheetID: spreadsheetID, rangeName: worksheet + "!A:Z"}, nil
}

// SpreadsheetIDFromURL extracts the ID from a standard Google Sheets URL.
// Credentials and query parameters are deliberately not part of the result.
func SpreadsheetIDFromURL(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "docs.google.com" {
		return "", fmt.Errorf("spreadsheet URL must be an HTTPS docs.google.com URL")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "d" && strings.TrimSpace(parts[i+1]) != "" {
			return parts[i+1], nil
		}
	}
	return "", fmt.Errorf("spreadsheet URL does not contain a spreadsheet ID")
}

func (s *GoogleSheetsSource) ReadRows(ctx context.Context) ([][]string, error) {
	values, err := s.service.Spreadsheets.Values.Get(s.spreadsheetID, s.rangeName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("read Google Sheet: %w", err)
	}
	rows := make([][]string, len(values.Values))
	for i, row := range values.Values {
		rows[i] = make([]string, len(row))
		for j, cell := range row {
			rows[i][j] = fmt.Sprint(cell)
		}
	}
	return rows, nil
}

type Diagnostic struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

type Snapshot struct {
	Sales    []history.Sale
	Rejected []Diagnostic
	SyncedAt time.Time
}

type Syncer struct {
	source RowsSource
	mu     sync.RWMutex
	latest Snapshot
}

func New(source RowsSource) *Syncer {
	return &Syncer{source: source}
}

// Sync validates all rows and atomically replaces the latest snapshot only
// with accepted rows. A bad row is diagnosed without discarding good rows.
func (s *Syncer) Sync(ctx context.Context) (Snapshot, error) {
	rows, err := s.source.ReadRows(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	if len(rows) == 0 {
		return Snapshot{}, fmt.Errorf("Google Sheet returned no header row")
	}
	header := csvLine(rows[0])
	var accepted []history.Sale
	var rejected []Diagnostic
	for index, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}
		sales, parseErr := history.ParseCSV(strings.NewReader(header + "\n" + csvLine(row) + "\n"))
		if parseErr != nil {
			rejected = append(rejected, Diagnostic{Row: index + 2, Reason: parseErr.Error()})
			continue
		}
		accepted = append(accepted, sales...)
	}
	snapshot := Snapshot{Sales: accepted, Rejected: rejected, SyncedAt: time.Now().UTC()}
	s.mu.Lock()
	s.latest = snapshot
	s.mu.Unlock()
	return snapshot, nil
}

func (s *Syncer) Current() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Snapshot{Sales: append([]history.Sale(nil), s.latest.Sales...), Rejected: append([]Diagnostic(nil), s.latest.Rejected...), SyncedAt: s.latest.SyncedAt}
}

func csvLine(values []string) string {
	data, _ := json.Marshal(values)
	// JSON strings and CSV fields share escaping for the simple sheet values we
	// accept; replace JSON array delimiters to produce a one-row CSV safely.
	var decoded []string
	_ = json.Unmarshal(data, &decoded)
	var builder strings.Builder
	for i, value := range decoded {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteByte('"')
		builder.WriteString(strings.ReplaceAll(value, "\"", "\"\""))
		builder.WriteByte('"')
	}
	return builder.String()
}
