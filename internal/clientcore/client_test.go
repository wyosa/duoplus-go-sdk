package clientcore

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, baseURL string, interval time.Duration) *Client {
	t.Helper()
	c, err := New("secret-key", Config{BaseURL: baseURL, Lang: "en", MinInterval: interval})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return c
}

func TestDoSendsHeadersAndDecodesData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/x" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("DuoPlus-API-Key"); got != "secret-key" {
			t.Errorf("api key header = %q", got)
		}
		if got := r.Header.Get("Lang"); got != "en" {
			t.Errorf("lang header = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content type = %q", got)
		}
		w.Write([]byte(`{"code":200,"data":{"id":"abc"},"message":"Success"}`))
	}))
	defer srv.Close()

	var out struct {
		ID string `json:"id"`
	}
	if err := newTestClient(t, srv.URL, time.Nanosecond).Do(context.Background(), "/api/v1/x", nil, &out); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if out.ID != "abc" {
		t.Fatalf("id = %q", out.ID)
	}
}

func TestDoReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":401,"data":null,"message":"login required"}`))
	}))
	defer srv.Close()

	err := newTestClient(t, srv.URL, time.Nanosecond).Do(context.Background(), "/api/v1/x", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %v", err)
	}
	if apiErr.Code != 401 || apiErr.Message != "login required" {
		t.Fatalf("unexpected api error %+v", apiErr)
	}
}

func TestDoRefusesRedirects(t *testing.T) {
	var leaked atomic.Bool
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("DuoPlus-API-Key") != "" {
			leaked.Store(true)
		}
		w.Write([]byte(`{"code":200,"data":{},"message":"ok"}`))
	}))
	defer evil.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, evil.URL+"/steal", http.StatusTemporaryRedirect)
	}))
	defer api.Close()

	for _, hc := range []*http.Client{nil, {Timeout: time.Second}} {
		c, err := New("secret-key", Config{BaseURL: api.URL, HTTPClient: hc, MinInterval: time.Nanosecond})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		err = c.Do(context.Background(), "/api/v1/x", nil, nil)
		if err == nil || !strings.Contains(err.Error(), "refusing to follow redirect") {
			t.Fatalf("expected redirect error, got %v", err)
		}
	}
	if leaked.Load() {
		t.Fatal("api key was sent to the redirect target")
	}
}

func TestNewDoesNotMutateCallerHTTPClient(t *testing.T) {
	hc := &http.Client{}
	if _, err := New("k", Config{HTTPClient: hc}); err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if hc.CheckRedirect != nil {
		t.Fatal("caller's http.Client was modified")
	}
}

func TestDoRejectsOversizedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":200,"data":"`))
		w.Write([]byte(strings.Repeat("a", maxResponseBytes)))
		w.Write([]byte(`"}`))
	}))
	defer srv.Close()

	err := newTestClient(t, srv.URL, time.Nanosecond).Do(context.Background(), "/api/v1/x", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected size limit error, got %v", err)
	}
}

func TestAPIErrorBodyIsTruncated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(strings.Repeat("x", 2*maxErrorBodyBytes)))
	}))
	defer srv.Close()

	err := newTestClient(t, srv.URL, time.Nanosecond).Do(context.Background(), "/api/v1/x", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %v", err)
	}
	if len(apiErr.Body) > maxErrorBodyBytes+len("...(truncated)") {
		t.Fatalf("body not truncated: %d bytes", len(apiErr.Body))
	}
}

func TestNewValidatesBaseURL(t *testing.T) {
	valid := map[string]string{
		"":                             DefaultBaseURL,
		"https://openapi.duoplus.cn/":  "https://openapi.duoplus.cn",
		"http://127.0.0.1:8080":        "http://127.0.0.1:8080",
		"http://localhost:8080/prefix": "http://localhost:8080/prefix",
		"http://[::1]:8080":            "http://[::1]:8080",
	}
	for in, want := range valid {
		c, err := New("k", Config{BaseURL: in})
		if err != nil {
			t.Errorf("New(%q) error = %v", in, err)
			continue
		}
		if c.baseURL != want {
			t.Errorf("New(%q) baseURL = %q, want %q", in, c.baseURL, want)
		}
	}

	invalid := []string{
		"http://openapi.duoplus.net",
		"ftp://openapi.duoplus.net",
		"openapi.duoplus.net",
		"https://user:pass@openapi.duoplus.net",
		"https://openapi.duoplus.net?x=1",
		"https://openapi.duoplus.net#frag",
		"https://",
	}
	for _, in := range invalid {
		_, err := New("k", Config{BaseURL: in})
		if err == nil {
			t.Errorf("New(%q) expected error", in)
			continue
		}
		if strings.Contains(err.Error(), "pass") {
			t.Errorf("New(%q) error leaks credentials: %v", in, err)
		}
	}
}

func TestRateLimitIsPerPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":200,"data":{},"message":"ok"}`))
	}))
	defer srv.Close()

	const interval = 300 * time.Millisecond
	c := newTestClient(t, srv.URL, interval)
	ctx := context.Background()

	start := time.Now()
	for _, path := range []string{"/a", "/b", "/c"} {
		if err := c.Do(ctx, path, nil, nil); err != nil {
			t.Fatalf("Do(%s) error = %v", path, err)
		}
	}
	if elapsed := time.Since(start); elapsed >= interval {
		t.Fatalf("different paths were throttled together: %v", elapsed)
	}

	start = time.Now()
	if err := c.Do(ctx, "/a", nil, nil); err != nil {
		t.Fatalf("Do(/a) error = %v", err)
	}
	if elapsed := time.Since(start); elapsed < interval/2 {
		t.Fatalf("same path was not throttled: %v", elapsed)
	}
}
