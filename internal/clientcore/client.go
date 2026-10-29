// Package clientcore sends rate-limited API requests and storage uploads.
package clientcore

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"reflect"
	"strconv"
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

	// A rate-limited request was not processed, so retrying it is safe.
	maxRateLimitRetries = 2
	maxRetryAfter       = time.Minute
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

	mu        sync.Mutex
	endpoints map[string]*endpoint
}

// endpoint paces the requests to one API path.
type endpoint struct {
	// next is the earliest start of the next request.
	next time.Time
	// sending is closed once the current request is written or fails; nil when idle.
	sending chan struct{}
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
		endpoints:   map[string]*endpoint{},
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

// Do retries only rate-limited requests (HTTP or API code 429), at most twice.
func (c *Client) Do(ctx context.Context, path string, requestBody any, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Serialize before taking a turn: a slow Marshaler must not let a later
	// request catch up with this one at the transport.
	payload := []byte("{}")
	if requestBody != nil {
		var err error
		payload, err = json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
	}

	var (
		status int
		header http.Header
		body   []byte
	)
	for retries := 0; ; retries++ {
		var err error
		status, header, body, err = c.send(ctx, path, payload)
		if err != nil {
			return err
		}
		if retries == maxRateLimitRetries || !rateLimited(status, body) {
			break
		}
		delay, ok := retryAfter(header.Get("Retry-After"))
		if !ok {
			break
		}
		if err := sleep(ctx, delay); err != nil {
			return err
		}
	}

	httpSuccess := status >= http.StatusOK && status < http.StatusMultipleChoices
	var apiResp envelope
	if err := json.Unmarshal(body, &apiResp); err != nil {
		if !httpSuccess {
			return &APIError{HTTPStatus: status, Body: truncate(body)}
		}
		return fmt.Errorf("decode response: %w", err)
	}

	if !httpSuccess || apiResp.Code != http.StatusOK {
		return &APIError{
			HTTPStatus: status,
			Code:       apiResp.Code,
			Message:    apiResp.Message,
			Body:       truncate(body),
		}
	}

	if out == nil || len(apiResp.Data) == 0 || string(apiResp.Data) == "null" {
		return nil
	}

	if err := decodeData(apiResp.Data, out); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}

	return nil
}

// send makes one request when the endpoint allows it and returns the response
// status, header, and body.
func (c *Client) send(ctx context.Context, path string, payload []byte) (int, http.Header, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Lang", c.lang)
	req.Header.Set("DuoPlus-API-Key", c.apiKey)

	written, err := c.waitTurn(ctx, path)
	if err != nil {
		return 0, nil, nil, err
	}
	trace := &httptrace.ClientTrace{WroteRequest: func(info httptrace.WroteRequestInfo) {
		if info.Err == nil {
			written()
		}
	}}
	resp, err := c.httpClient.Do(req.WithContext(httptrace.WithClientTrace(ctx, trace)))
	// Custom transports may not report the write, and failed requests end the turn too.
	written()
	if err != nil {
		return 0, nil, nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return 0, nil, nil, fmt.Errorf("read response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return 0, nil, nil, fmt.Errorf("read response: body exceeds %d bytes", maxResponseBytes)
	}
	return resp.StatusCode, resp.Header, body, nil
}

// rateLimited reports an HTTP or API code 429.
func rateLimited(status int, body []byte) bool {
	var apiResp struct {
		Code int `json:"code"`
	}
	return status == http.StatusTooManyRequests ||
		json.Unmarshal(body, &apiResp) == nil && apiResp.Code == http.StatusTooManyRequests
}

// retryAfter parses a Retry-After value in seconds or as an HTTP date. It
// reports false when the server asks to wait longer than maxRetryAfter.
func retryAfter(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	var delay time.Duration
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds > int(maxRetryAfter/time.Second) {
			return 0, false
		}
		delay = time.Duration(seconds) * time.Second
	} else if date, err := http.ParseTime(value); err == nil {
		delay = time.Until(date)
	}
	return delay, delay <= maxRetryAfter
}

// decodeData decodes data into out. The API sends [] for some empty objects,
// so when decoding fails on an array, the empty arrays that out decodes as
// maps or structs are replaced with {} and decoding is retried.
func decodeData(data []byte, out any) error {
	err := json.Unmarshal(data, out)
	var typeErr *json.UnmarshalTypeError
	if !errors.As(err, &typeErr) || typeErr.Value != "array" {
		return err
	}
	target := reflect.ValueOf(out)
	if target.Kind() != reflect.Pointer || target.IsNil() {
		return err
	}

	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return err
	}
	value, replaced := emptyObjects(value, target.Type().Elem())
	if !replaced {
		return err
	}
	normalized, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		return err
	}

	fresh := reflect.New(target.Type().Elem())
	if err := json.Unmarshal(normalized, fresh.Interface()); err != nil {
		return err
	}
	target.Elem().Set(fresh.Elem())
	return nil
}

var unmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()

// emptyObjects replaces each empty array in v that t decodes as a map or a
// struct with an empty object. Types with their own UnmarshalJSON are skipped.
func emptyObjects(v any, t reflect.Type) (any, bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if reflect.PointerTo(t).Implements(unmarshalerType) {
		return v, false
	}

	replaced := false
	switch v := v.(type) {
	case []any:
		switch t.Kind() {
		case reflect.Map, reflect.Struct:
			if len(v) == 0 {
				return map[string]any{}, true
			}
		case reflect.Slice, reflect.Array:
			for i := range v {
				var r bool
				v[i], r = emptyObjects(v[i], t.Elem())
				replaced = replaced || r
			}
		}
	case map[string]any:
		for key, item := range v {
			if field, ok := fieldType(t, key); ok {
				var r bool
				v[key], r = emptyObjects(item, field)
				replaced = replaced || r
			}
		}
	}
	return v, replaced
}

// fieldType returns the type that the object key decodes into within t,
// including fields promoted from embedded structs.
func fieldType(t reflect.Type, key string) (reflect.Type, bool) {
	switch t.Kind() {
	case reflect.Map:
		return t.Elem(), true
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			if name == "-" {
				continue
			}
			embedded := field.Type
			if embedded.Kind() == reflect.Pointer {
				embedded = embedded.Elem()
			}
			if field.Anonymous && name == "" && embedded.Kind() == reflect.Struct {
				if found, ok := fieldType(embedded, key); ok {
					return found, true
				}
				continue
			}
			if !field.IsExported() {
				continue
			}
			if name == "" {
				name = field.Name
			}
			if strings.EqualFold(name, key) {
				return field.Type, true
			}
		}
	}
	return nil, false
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
	// OSS stores the object but answers 203 when the upload callback fails, so
	// the file never reaches the cloud drive.
	if resp.StatusCode < 200 || resp.StatusCode > 299 || resp.StatusCode == http.StatusNonAuthoritativeInfo {
		return &APIError{HTTPStatus: resp.StatusCode, Message: storageErrorMessage(respBody), Body: truncate(respBody)}
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

// storageErrorMessage returns "Code: Message" from an OSS XML error body.
func storageErrorMessage(body []byte) string {
	var e struct {
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	}
	if xml.Unmarshal(body, &e) != nil || e.Code == "" {
		return ""
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

// waitTurn enforces the documented limit of 1 QPS per interface. The interval
// starts once the previous request is written, so slow connection setup cannot
// shorten the gap the server sees. Call the returned function when the request
// is written or has failed; later calls are ignored.
func (c *Client) waitTurn(ctx context.Context, path string) (func(), error) {
	for {
		c.mu.Lock()
		if err := ctx.Err(); err != nil {
			c.mu.Unlock()
			return nil, err
		}
		e := c.endpoints[path]
		if e == nil {
			e = &endpoint{}
			c.endpoints[path] = e
		}
		sending := e.sending
		wait := time.Until(e.next)
		if sending == nil && wait <= 0 {
			done := make(chan struct{})
			e.sending = done
			c.mu.Unlock()
			var once sync.Once
			return func() {
				once.Do(func() {
					c.mu.Lock()
					e.next = time.Now().Add(c.minInterval)
					e.sending = nil
					c.mu.Unlock()
					close(done)
				})
			}, nil
		}
		c.mu.Unlock()

		// Waiting callers do not reserve future slots. After waking, check
		// again under the lock in case another caller took this turn.
		if sending != nil {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-sending:
			}
		} else if err := sleep(ctx, wait); err != nil {
			return nil, err
		}
	}
}

// sleep waits for d unless ctx is done first.
func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
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
