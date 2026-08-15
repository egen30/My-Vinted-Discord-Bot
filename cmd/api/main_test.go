package main

import (
	"testing"

	"github.com/2spy/vinted-discord-bot/pkg/models"
)

func TestValidateSearchAcceptsVintedHTTPSURL(t *testing.T) {
	if err := validateSearch(models.Search{Name: "Nike 42", URL: "https://www.vinted.de/catalog?search_text=nike"}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSearchRejectsNonVintedURL(t *testing.T) {
	if err := validateSearch(models.Search{Name: "bad", URL: "https://example.com/search"}); err == nil {
		t.Fatal("expected unsupported host error")
	}
}
