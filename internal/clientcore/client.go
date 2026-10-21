package clientcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	DefaultBaseURL     = "https://openapi.duoplus.net"
	DefaultIntlBaseURL = DefaultBaseURL
	DefaultCNBaseURL   = "https://openapi.duoplus.cn"
	DefaultLanguage    = "en"
	defaultTimeout     = 30 * time.Second
	defaultMinInterval = time.Second

	maxResponseBytes  = 10 << 20
	maxErrorBodyBytes = 4 << 10
)

type Config struct {
	BaseURL     string
	Lang        string
	HTTPClient  *http.Client
	MinInterval time.Duration
}

type Client struct {
	apiKey      string
	baseURL     string
	lang        string
	httpClient  *http.Client
	minInterval time.Duration

	mu          sync.Mutex
	nextRequest map[string]time.Time
}

type APIError struct {
	HTTPStatus int
	Code       int
	Message    string
	Body       string
}

type envelope struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

func New(apiKey string, cfg Config) (*Client, error) {
	baseURL, err := normalizeBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	lang := strings.TrimSpace(cfg.Lang)
	if lang == "" {
		lang = DefaultLanguage
	}

	minInterval := cfg.MinInterval
	if minInterval <= 0 {
		minInterval = defaultMinInterval
	}

	return &Client{
		apiKey:      apiKey,
		baseURL:     baseURL,
		lang:        lang,
		httpClient:  withoutRedirects(cfg.HTTPClient),
		minInterval: minInterval,
		nextRequest: map[string]time.Time{},
	}, nil
}

func (e *APIError) Error() string {
	parts := make([]string, 0, 3)
	if e.HTTPStatus > 0 {
		parts = append(parts, fmt.Sprintf("http_status=%d", e.HTTPStatus))
	}
	if e.Code > 0 {
		parts = append(parts, fmt.Sprintf("code=%d", e.Code))
	}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	if len(parts) == 0 && e.Body != "" {
		parts = append(parts, e.Body)
	}
	if len(parts) == 0 {
		return "duoplus api error"
	}
	return "duoplus api error: " + strings.Join(parts, ", ")
}

func (c *Client) Do(ctx context.Context, path string, requestBody any, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := c.waitTurn(ctx, path); err != nil {
		return err
	}

	payload := []byte("{}")
	if requestBody != nil {
		var err error
		payload, err = json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Lang", c.lang)
	req.Header.Set("DuoPlus-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return fmt.Errorf("read response: body exceeds %d bytes", maxResponseBytes)
	}

	var apiResp envelope
	if err := json.Unmarshal(body, &apiResp); err != nil {
		if resp.StatusCode >= http.StatusBadRequest {
			return &APIError{HTTPStatus: resp.StatusCode, Body: truncate(body)}
		}
		return fmt.Errorf("decode response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest || apiResp.Code != http.StatusOK {
		return &APIError{
			HTTPStatus: resp.StatusCode,
			Code:       apiResp.Code,
			Message:    apiResp.Message,
			Body:       truncate(body),
		}
	}

	if out == nil || len(apiResp.Data) == 0 || string(apiResp.Data) == "null" {
		return nil
	}

	if err := json.Unmarshal(apiResp.Data, out); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}

	return nil
}

// Upload sends body to a presigned storage URL returned by the API. The API
// key is not sent, and the client timeout is lifted so that large files are
// bounded only by ctx. size must be the exact body length.
func (c *Client) Upload(ctx context.Context, method, signedURL string, headers map[string]string, body io.Reader, size int64) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if method == "" {
		method = http.MethodPut
	}
	if method != http.MethodPut && method != http.MethodPost {
		return fmt.Errorf("upload: unsupported method %q", method)
	}

	u, err := url.Parse(signedURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return errors.New("upload: invalid signed url")
	}

	if size == 0 {
		body = http.NoBody
	}
	req, err := http.NewRequestWithContext(ctx, method, signedURL, body)
	if err != nil {
		return errors.New("upload: build request failed")
	}
	req.ContentLength = size
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	hc := *c.httpClient
	hc.Timeout = 0
	resp, err := hc.Do(req)
	if err != nil {
		// url.Error would include the signed URL, which is a credential.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return fmt.Errorf("upload: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("upload: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &APIError{HTTPStatus: resp.StatusCode, Body: truncate(respBody)}
	}

	var apiResp envelope
	if json.Unmarshal(respBody, &apiResp) == nil && apiResp.Code != 0 && apiResp.Code != http.StatusOK {
		return &APIError{HTTPStatus: resp.StatusCode, Code: apiResp.Code, Message: apiResp.Message, Body: truncate(respBody)}
	}
	return nil
}

// waitTurn enforces the documented limit of 1 QPS per interface.
func (c *Client) waitTurn(ctx context.Context, path string) error {
	c.mu.Lock()
	now := time.Now()
	when := c.nextRequest[path]
	if when.Before(now) {
		when = now
	}
	c.nextRequest[path] = when.Add(c.minInterval)
	c.mu.Unlock()

	wait := time.Until(when)
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func normalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return DefaultBaseURL, nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return "", fmt.Errorf("invalid base url: %w", err)
	}
	if u.User != nil {
		return "", errors.New("invalid base url: credentials in url are not allowed")
	}
	if u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("invalid base url %q: expected scheme://host[/path]", raw)
	}

	switch u.Scheme {
	case "https":
	case "http":
		if !isLoopback(u.Hostname()) {
			return "", fmt.Errorf("invalid base url %q: plain http is only allowed for loopback hosts", raw)
		}
	default:
		return "", fmt.Errorf("invalid base url %q: scheme must be https", raw)
	}

	return raw, nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// withoutRedirects copies hc and disables redirects: net/http forwards custom
// headers such as DuoPlus-API-Key to any redirect target, including other hosts.
func withoutRedirects(hc *http.Client) *http.Client {
	var c http.Client
	if hc != nil {
		c = *hc
	} else {
		c.Timeout = defaultTimeout
	}
	c.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		return fmt.Errorf("refusing to follow redirect to %s", req.URL.Redacted())
	}
	return &c
}

func truncate(body []byte) string {
	if len(body) <= maxErrorBodyBytes {
		return string(body)
	}
	return string(body[:maxErrorBodyBytes]) + "...(truncated)"
}
