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
	"unicode/utf8"
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
	maxErrorTextBytes = 4 << 10
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

// APIError reports a non-2xx HTTP status or non-200 API code.
type APIError struct {
	// HTTPStatus is the HTTP response status, independently of Code.
	HTTPStatus int
	// Code is the DuoPlus envelope code, or zero when the envelope is unavailable.
	Code int
	// Message is the original, untruncated server message.
	Message string
	// Body contains at most the first 4 KiB of the raw response plus a truncation marker.
	Body string
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

// Error returns at most 4 KiB of valid UTF-8. Diagnostic fields are not changed or redacted.
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
	return limitErrorText("duoplus api error: " + strings.Join(parts, ", "))
}

func limitErrorText(text string) string {
	text = strings.ToValidUTF8(text, "\uFFFD")
	if len(text) <= maxErrorTextBytes {
		return text
	}
	const suffix = "...(truncated)"
	end := maxErrorTextBytes - len(suffix)
	for !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + suffix
}

func (c *Client) Do(ctx context.Context, path string, requestBody any, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
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

	// Serialize and build the request before taking a turn: a slow Marshaler
	// must not let a later request catch up with this one at the transport.
	if err := c.waitTurn(ctx, path); err != nil {
		return err
	}

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

	httpSuccess := resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
	var apiResp envelope
	if err := json.Unmarshal(body, &apiResp); err != nil {
		if !httpSuccess {
			return &APIError{HTTPStatus: resp.StatusCode, Body: truncate(body)}
		}
		return fmt.Errorf("decode response: %w", err)
	}

	if !httpSuccess || apiResp.Code != http.StatusOK {
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
		// Redirect parsing and custom transports may also include URLs in
		// their errors. Keep the cause available without printing credentials.
		return &uploadRequestError{err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("upload: read response: %w", err)
	}
	if len(respBody) > maxResponseBytes {
		return fmt.Errorf("upload: read response: body exceeds %d bytes", maxResponseBytes)
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

type uploadRequestError struct {
	err error
}

func (e *uploadRequestError) Error() string { return "upload: request failed" }
func (e *uploadRequestError) Unwrap() error { return e.err }

// waitTurn enforces the documented limit of 1 QPS per interface.
func (c *Client) waitTurn(ctx context.Context, path string) error {
	for {
		c.mu.Lock()
		if err := ctx.Err(); err != nil {
			c.mu.Unlock()
			return err
		}
		now := time.Now()
		wait := c.nextRequest[path].Sub(now)
		if wait <= 0 {
			c.nextRequest[path] = now.Add(c.minInterval)
			c.mu.Unlock()
			return nil
		}
		c.mu.Unlock()

		// Waiting callers do not reserve future slots. After waking, check
		// again under the lock in case another caller took this turn.
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
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
	if u.Hostname() == "" || u.RawQuery != "" || u.ForceQuery || strings.Contains(raw, "#") {
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
	c.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return errors.New("refusing to follow redirect")
	}
	return &c
}

func truncate(body []byte) string {
	if len(body) <= maxErrorBodyBytes {
		return string(body)
	}
	return string(body[:maxErrorBodyBytes]) + "...(truncated)"
}
