package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/2spy/vinted-discord-bot/pkg/models"
)

func TestSendListingIncludesFinancialFieldsAndImages(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	notifier := &DiscordNotifier{webhookURL: server.URL, client: server.Client()}
	updated := time.Now().Add(-30 * time.Minute)
	err := notifier.SendListing(context.Background(), models.Opportunity{
		Item:                models.Item{Brand: "Nike", Title: "P-6000", URL: "https://vinted.test/item", Price: 15, Currency: "EUR", Size: "42", Description: "Aucun défaut", UpdatedAt: &updated, Seller: models.Seller{Username: "jennajfr", Country: "FR", Rating: 5, ReviewCount: 5}, ImageURL: "https://img.test/1", ImageURLs: []string{"https://img.test/1", "https://img.test/2"}},
		ExpectedResaleCents: 3500, ExpectedProfitCents: 2000, MaximumPurchaseCents: 2200, ROIPercent: 133.3, Condition: "good",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"🇫🇷 P-6000", "👤 **jennajfr**", "Aucun défaut", "See more on Vinted", "📅 Updated", "30 minutes ago", "📏 Size", "🏷️ Brand", "📦 Condition", "🌟 Rating", "⭐️⭐️⭐️⭐️⭐️ (5)", "💰 Price", "35.00 €", "https://img.test/1"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("payload missing %q: %s", expected, body)
		}
	}
	if strings.Contains(body, "Expected profit") {
		t.Fatal("notification should keep the main card focused on the listing details")
	}
}

func TestSendListingUsesDiscordMediaGalleryAttachments(t *testing.T) {
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("fake-jpeg-" + r.URL.Path))
	}))
	defer imageServer.Close()
	var attachmentNames []string
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse webhook multipart form: %v", err)
			return
		}
		for _, files := range r.MultipartForm.File {
			for _, file := range files {
				attachmentNames = append(attachmentNames, file.Filename)
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer webhook.Close()

	notifier := &DiscordNotifier{webhookURL: webhook.URL, client: webhook.Client()}
	err := notifier.SendListing(context.Background(), models.Opportunity{Item: models.Item{
		Title:     "Gallery test",
		ImageURLs: []string{imageServer.URL + "/one", imageServer.URL + "/two", imageServer.URL + "/three"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(attachmentNames) != 3 {
		t.Fatalf("got %d attachments, want 3 (%v)", len(attachmentNames), attachmentNames)
	}
	sort.Strings(attachmentNames)
	for i, name := range attachmentNames {
		want := "vinted-photo-" + string(rune('1'+i)) + ".jpg"
		if name != want {
			t.Errorf("attachment %d = %q, want %q", i, name, want)
		}
	}
}
