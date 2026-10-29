package clientcore

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCanceledRequestsDoNotReserveTurns(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":200,"data":{}}`)
	}))
	defer srv.Close()
	c := newTestClient(t, srv.URL, 50*time.Millisecond)
	if err := c.Do(context.Background(), "/a", nil, nil); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for i := 0; i < 20; i++ {
		if err := c.Do(canceled, "/a", nil, nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled request: %v", err)
		}
	}
	// Also cancel a call that has already started waiting for its turn.
	waiting, cancelWaiting := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancelWaiting()
	if err := c.Do(waiting, "/a", nil, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waiting request: %v", err)
	}
	healthy, cancelHealthy := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancelHealthy()
	if err := c.Do(healthy, "/a", nil, nil); err != nil {
		t.Fatalf("canceled calls delayed a healthy request: %v", err)
	}
}

type blockingMarshaler struct {
	started chan struct{}
	release chan struct{}
}

func (m blockingMarshaler) MarshalJSON() ([]byte, error) {
	close(m.started)
	<-m.release
	return []byte(`{}`), nil
}

func TestRateLimitStartsAfterSerialization(t *testing.T) {
	const interval = 60 * time.Millisecond
	arrivals := make(chan time.Time, 2)
	c, err := New("key", Config{MinInterval: interval, HTTPClient: &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			arrivals <- time.Now()
			return &http.Response{StatusCode: 200, Header: make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{"code":200,"data":{}}`)), Request: r}, nil
		}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	m := blockingMarshaler{started: make(chan struct{}), release: make(chan struct{})}
	finished := make(chan error, 1)
	go func() { finished <- c.Do(ctx, "/a", m, nil) }()
	<-m.started
	secondErr := c.Do(ctx, "/a", nil, nil)
	close(m.release)
	firstErr := <-finished
	if firstErr != nil || secondErr != nil {
		t.Fatalf("requests failed: %v, %v", firstErr, secondErr)
	}
	first, second := <-arrivals, <-arrivals
	if gap := second.Sub(first); gap < interval*4/5 {
		t.Fatalf("transport calls only %v apart, interval is %v", gap, interval)
	}
}

func TestConcurrentWaitersRespectInterval(t *testing.T) {
	const count, interval = 5, 30 * time.Millisecond
	arrivals := make(chan time.Time, count)
	c, err := New("key", Config{MinInterval: interval, HTTPClient: &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			arrivals <- time.Now()
			return &http.Response{StatusCode: 200, Header: make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{"code":200}`)), Request: r}, nil
		}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	finished := make(chan error, count)
	for i := 0; i < count; i++ {
		go func() { finished <- c.Do(ctx, "/a", nil, nil) }()
	}
	for i := 0; i < count; i++ {
		if err := <-finished; err != nil {
			t.Fatal(err)
		}
	}
	previous := <-arrivals
	for i := 1; i < count; i++ {
		now := <-arrivals
		if gap := now.Sub(previous); gap < interval*4/5 {
			t.Fatalf("concurrent requests only %v apart", gap)
		}
		previous = now
	}
}

func TestDoRejectsNonSuccessHTTPStatus(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"multiple choices", 300, `{"code":200,"data":{}}`},
		{"not modified", 304, ""},
		{"redirect with invalid JSON", 302, "redirect"},
		{"server error", 500, `{"code":200,"data":{}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()
			err := newTestClient(t, srv.URL, time.Nanosecond).Do(context.Background(), "/a", nil, nil)
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.HTTPStatus != tc.status {
				t.Fatalf("expected HTTP %d error, got %v", tc.status, err)
			}
		})
	}
}

type embeddedInfo struct {
	Info map[string]string `json:"info"`
}

func TestDoAcceptsEmptyArraysForObjects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{
			"/top":    `[ ]`,
			"/nested": `{"id":"p1","proxy":[],"headers":[],"info":[],"list":[[],{"extra":[],"tags":[]}]}`,
			"/other":  `{"proxy":[],"count":"x"}`,
			"/items":  `[1]`,
		}[r.URL.Path]
		io.WriteString(w, `{"code":200,"data":`+data+`}`)
	}))
	defer srv.Close()
	c := newTestClient(t, srv.URL, time.Nanosecond)
	ctx := context.Background()

	var object struct {
		ID string `json:"id"`
	}
	values := map[string]string{}
	var list []string
	for _, out := range []any{&object, &values, &list} {
		if err := c.Do(ctx, "/top", nil, out); err != nil {
			t.Fatalf("%T: %v", out, err)
		}
	}

	var nested struct {
		embeddedInfo
		ID    string `json:"id"`
		Proxy struct {
			Host string `json:"host"`
		} `json:"proxy"`
		Headers map[string]string `json:"headers"`
		List    []struct {
			Extra map[string]any `json:"extra"`
			Tags  []string       `json:"tags"`
		} `json:"list"`
	}
	if err := c.Do(ctx, "/nested", nil, &nested); err != nil {
		t.Fatalf("nested empty arrays: %v", err)
	}
	if nested.ID != "p1" || len(nested.List) != 2 || nested.List[1].Tags == nil {
		t.Fatalf("lost decoded fields: %+v", nested)
	}

	var other struct {
		Proxy struct{} `json:"proxy"`
		Count int      `json:"count"`
	}
	var number int
	for path, out := range map[string]any{"/other": &other, "/items": &object, "/top": &number} {
		if err := c.Do(ctx, path, nil, out); err == nil {
			t.Errorf("%s: decoded a mismatched value into %T", path, out)
		}
	}
}

func TestRateLimitCountsFromWrittenRequest(t *testing.T) {
	const interval = 300 * time.Millisecond
	arrivals := make(chan time.Time, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		arrivals <- time.Now()
		io.WriteString(w, `{"code":200,"data":{}}`)
	}))
	defer srv.Close()
	var dialer net.Dialer
	transport := &http.Transport{DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		// Slow connection setup delays when the first request reaches the server.
		time.Sleep(interval / 2)
		return dialer.DialContext(ctx, network, addr)
	}}
	defer transport.CloseIdleConnections()
	c, err := New("key", Config{BaseURL: srv.URL, MinInterval: interval, HTTPClient: &http.Client{Transport: transport}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := c.Do(context.Background(), "/a", nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	first, second := <-arrivals, <-arrivals
	if gap := second.Sub(first); gap < interval*9/10 {
		t.Fatalf("server saw requests %v apart, interval is %v", gap, interval)
	}
}

func TestDoRetriesRateLimitedRequests(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		body       string
		retryAfter string
		limited    int32
		calls      int32
	}{
		{"http status", http.StatusTooManyRequests, `{"message":"Too Many Attempts."}`, "0", 2, 3},
		{"api code", http.StatusOK, `{"code":429,"message":"too frequent"}`, "", 2, 3},
		{"gives up", http.StatusTooManyRequests, `{"code":429,"message":"too frequent"}`, "", 3, 3},
		{"long retry-after", http.StatusTooManyRequests, `{"code":429,"message":"too frequent"}`, "3600", 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if calls.Add(1) <= tc.limited {
					if tc.retryAfter != "" {
						w.Header().Set("Retry-After", tc.retryAfter)
					}
					w.WriteHeader(tc.status)
					io.WriteString(w, tc.body)
					return
				}
				io.WriteString(w, `{"code":200,"data":{}}`)
			}))
			defer srv.Close()
			err := newTestClient(t, srv.URL, time.Nanosecond).Do(context.Background(), "/a", nil, nil)
			if calls.Load() != tc.calls {
				t.Fatalf("sent %d requests, want %d", calls.Load(), tc.calls)
			}
			var apiErr *APIError
			if tc.limited < tc.calls {
				if err != nil {
					t.Fatalf("retry failed: %v", err)
				}
			} else if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusTooManyRequests {
				t.Fatalf("expected the rate-limit error, got %v", err)
			}
		})
	}
}

func TestUploadErrorsHideRedirectCredentials(t *testing.T) {
	for _, location := range []string{
		"/redirected?Signature=dummy-secret",
		"https://storage.example.test/file?Signature=dummy-secret",
		"/invalid%zz?Signature=dummy-secret",
	} {
		t.Run(location, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/object" {
					t.Error("upload followed a redirect")
				}
				w.Header().Set("Location", location)
				w.WriteHeader(http.StatusTemporaryRedirect)
			}))
			defer srv.Close()
			err := newTestClient(t, srv.URL, time.Nanosecond).Upload(context.Background(), "PUT",
				srv.URL+"/object?Signature=original-secret", nil, strings.NewReader("x"), 1)
			if err == nil || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "Signature") {
				t.Fatalf("upload error is missing or leaks credentials: %v", err)
			}
		})
	}
}

func TestUploadErrorPreservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := newTestClient(t, "http://127.0.0.1:1", time.Nanosecond).Upload(ctx, "PUT",
		"http://127.0.0.1:1/object?Signature=secret", nil, strings.NewReader("x"), 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was lost: %v", err)
	}
}

func TestUploadRejectsOversizedErrorCallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":500,"message":"`+strings.Repeat("x", maxResponseBytes)+`"}`)
	}))
	defer srv.Close()
	err := newTestClient(t, srv.URL, time.Nanosecond).Upload(context.Background(), "PUT", srv.URL,
		nil, strings.NewReader("x"), 1)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected response size error, got %v", err)
	}
}
