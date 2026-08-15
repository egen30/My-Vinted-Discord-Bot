package vinted

import "testing"

func TestJobFromSearchURL(t *testing.T) {
	job, err := JobFromSearchURL("https://www.vinted.de/catalog?search_text=nike%20p-6000&catalog_ids=1242&size_ids=782,783&price_to=22")
	if err != nil {
		t.Fatal(err)
	}
	if job.Query != "nike p-6000" || len(job.SizeIDs) != 2 || job.MaxPrice != 22 || job.Domain != "https://www.vinted.de" {
		t.Fatalf("unexpected job: %+v", job)
	}
}

func TestJobFromSearchURLRejectsUnsupportedHost(t *testing.T) {
	if _, err := JobFromSearchURL("https://example.com/catalog"); err == nil {
		t.Fatal("expected unsupported host error")
	}
}
