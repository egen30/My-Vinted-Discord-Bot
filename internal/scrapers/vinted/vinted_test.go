package vinted

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	http "github.com/bogdanfinn/fhttp"
)

type fakeHTTPClient struct {
	responses []*http.Response
	errors    []error
	calls     int
}

func (f *fakeHTTPClient) Do(*http.Request) (*http.Response, error) {
	index := f.calls
	f.calls++
	var response *http.Response
	if index < len(f.responses) {
		response = f.responses[index]
	}
	var err error
	if index < len(f.errors) {
		err = f.errors[index]
	}
	return response, err
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestReadResponseBodyRejectsNonSuccess(t *testing.T) {
	_, err := readResponseBody(response(http.StatusForbidden, "blocked"))
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestReadResponseBodyReadsSuccess(t *testing.T) {
	body, err := readResponseBody(response(http.StatusOK, `{"items":[]}`))
	if err != nil {
		t.Fatalf("readResponseBody returned error: %v", err)
	}
	if string(body) != `{"items":[]}` {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestDoWithRetryRetriesTransientFailures(t *testing.T) {
	client := &fakeHTTPClient{
		responses: []*http.Response{
			response(http.StatusServiceUnavailable, "busy"),
			response(http.StatusTooManyRequests, "slow down"),
			response(http.StatusOK, "ok"),
		},
	}
	req, err := http.NewRequest("GET", "https://example.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := doWithRetry(context.Background(), client, req)
	if err != nil {
		t.Fatalf("doWithRetry returned error: %v", err)
	}
	if client.calls != 3 || got.StatusCode != http.StatusOK {
		t.Fatalf("expected 3 calls and 200 response, got %d calls and %d", client.calls, got.StatusCode)
	}
	_ = got.Body.Close()
}

func TestDoWithRetryStopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &fakeHTTPClient{errors: []error{errors.New("network")}}
	req, err := http.NewRequest("GET", "https://example.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doWithRetry(ctx, client, req); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if client.calls != 0 {
		t.Fatalf("expected no request after cancellation, got %d", client.calls)
	}
}
