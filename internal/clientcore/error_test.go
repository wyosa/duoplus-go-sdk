package clientcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestAPIErrorFormatting(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  APIError
		want string
	}{
		{"empty", APIError{}, "duoplus api error"},
		{"status and code", APIError{HTTPStatus: 429, Code: 403, Message: "quota"}, "duoplus api error: http_status=429, code=403, quota"},
		{"body fallback", APIError{Body: "storage failed"}, "duoplus api error: storage failed"},
		{"exact limit", APIError{Message: strings.Repeat("x", maxErrorTextBytes-len("duoplus api error: "))}, ""},
		{"long ASCII", APIError{Message: strings.Repeat("x", 64<<10)}, ""},
		{"long UTF-8", APIError{Message: strings.Repeat("ошибка🚀", 4096)}, ""},
		{"long body", APIError{Body: strings.Repeat("界", 4096)}, ""},
		{"invalid UTF-8", APIError{Message: "invalid\xfftext"}, "duoplus api error: invalid\uFFFDtext"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.err
			got := tc.err.Error()
			if len(got) > maxErrorTextBytes || !utf8.ValidString(got) {
				t.Fatalf("invalid error text: bytes=%d, valid UTF-8=%v", len(got), utf8.ValidString(got))
			}
			if tc.want != "" && got != tc.want {
				t.Fatalf("Error() = %q, want %q", got, tc.want)
			}
			if len(tc.err.Message)+len(tc.err.Body) > maxErrorTextBytes && !strings.HasSuffix(got, "...(truncated)") {
				t.Fatal("truncated error has no marker")
			}
			if tc.err != before {
				t.Fatal("formatting changed the diagnostic fields")
			}
		})
	}
}

func TestLongServerMessageRemainsInspectable(t *testing.T) {
	message := strings.Repeat("Ошибка сервера 🚀 ", 4096)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{"code": 429, "message": message})
	}))
	defer srv.Close()
	err := newTestClient(t, srv.URL, time.Nanosecond).Do(context.Background(), "/api/v1/test", nil, nil)
	if err == nil || len(err.Error()) > maxErrorTextBytes || !utf8.ValidString(err.Error()) {
		t.Fatal("server error text was not bounded safely")
	}
	var apiErr *APIError
	if !errors.As(fmt.Errorf("operation: %w", err), &apiErr) {
		t.Fatalf("APIError was lost: %v", err)
	}
	if apiErr.HTTPStatus != 429 || apiErr.Code != 429 || apiErr.Message != message {
		t.Fatal("original server diagnostics were not preserved")
	}
}
