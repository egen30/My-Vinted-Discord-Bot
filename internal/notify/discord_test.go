package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2spy/vinted-discord-bot/pkg/models"
)

func TestSendOpportunityIncludesFinancialFieldsAndImages(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	notifier := &DiscordNotifier{webhookURL: server.URL, client: server.Client()}
	err := notifier.SendOpportunity(context.Background(), models.Opportunity{
		Item:                models.Item{Brand: "Nike", Title: "P-6000", URL: "https://vinted.test/item", Price: 15, Currency: "EUR", Size: "42", ImageURL: "https://img.test/1", ImageURLs: []string{"https://img.test/1", "https://img.test/2"}},
		ExpectedResaleCents: 3500, ExpectedProfitCents: 2000, MaximumPurchaseCents: 2200, ROIPercent: 133.3, Condition: "good",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Expected resale", "35.00 EUR", "Expected profit", "20.00 EUR", "https://img.test/2"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("payload missing %q: %s", expected, body)
		}
	}
}
