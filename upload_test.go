package duoplus_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/clouddisk"
)

func TestUploadFile(t *testing.T) {
	for _, tc := range []struct{ name, contents, override string }{
		{"default name", "APK contents", ""},
		{"explicit name", "APK contents", "renamed.apk"},
		{"empty file", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "local.apk")
			if err := os.WriteFile(path, []byte(tc.contents), 0600); err != nil {
				t.Fatal(err)
			}
			wantName := tc.override
			if wantName == "" {
				wantName = "local.apk"
			}
			var calls atomic.Int32
			var srv *httptest.Server
			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				switch r.URL.Path {
				case "/api/v1/cloudDisk/signedUrl":
					var req clouddisk.SignedURLRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Error(err)
					}
					if req.Name != wantName || req.IsApp != 1 || req.Pkg != "com.example.app" {
						t.Errorf("unexpected signing request: %+v", req)
					}
					json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": clouddisk.SignedURLResponse{
						Method: "PUT", SignedURL: srv.URL + "/storage?Signature=fixture", Name: wantName,
						Headers: map[string]string{"x-oss-callback": "callback", "x-oss-callback-var": "variables"},
					}})
				case "/storage":
					data, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
					}
					if r.Method != "PUT" || string(data) != tc.contents || r.ContentLength != int64(len(tc.contents)) {
						t.Errorf("wrong file upload: %s length=%d body=%q", r.Method, r.ContentLength, data)
					}
					if r.Header.Get("DuoPlus-API-Key") != "" {
						t.Error("API key sent to storage")
					}
					if r.Header.Get("x-oss-callback") != "callback" || r.Header.Get("x-oss-callback-var") != "variables" {
						t.Error("missing callback headers")
					}
					fmt.Fprint(w, `{"code":200,"message":"Success"}`)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
				}
			}))
			defer srv.Close()
			client, err := duoplus.NewClient("fixture-key", duoplus.WithBaseURL(srv.URL))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			got, err := client.CloudDisk.UploadFile(ctx, path, clouddisk.SignedURLRequest{Name: tc.override, IsApp: 1, Pkg: "com.example.app"})
			if err != nil || got.Name != wantName || calls.Load() != 2 {
				t.Fatalf("UploadFile() = %+v, %v; requests=%d", got, err, calls.Load())
			}
		})
	}
}

func TestUploadFileRejectsLocalErrorsBeforeCallingAPI(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	defer srv.Close()
	client, err := duoplus.NewClient("fixture-key", duoplus.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, path := range []string{dir, filepath.Join(dir, "missing.apk")} {
		if _, err := client.CloudDisk.UploadFile(context.Background(), path, clouddisk.SignedURLRequest{}); err == nil {
			t.Errorf("accepted %s", path)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("called the API for an invalid local file")
	}
}

func TestUploadPropagatesStorageCallbackFailure(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		code    int
		message string
	}{
		{"callback error", http.StatusOK, `{"code":500,"message":"callback failed"}`, 500, "callback failed"},
		// OSS keeps the object but reports a failed callback with 203.
		{"oss callback failure", http.StatusNonAuthoritativeInfo,
			`<?xml version="1.0" encoding="UTF-8"?><Error><Code>CallbackFailed</Code><Message>Error status : 502.</Message></Error>`,
			0, "CallbackFailed: Error status : 502."},
		{"oss error", http.StatusForbidden,
			`<?xml version="1.0" encoding="UTF-8"?><Error><Code>SignatureDoesNotMatch</Code><Message>The request signature does not match.</Message></Error>`,
			0, "SignatureDoesNotMatch: The request signature does not match."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var srv *httptest.Server
			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/cloudDisk/signedUrl" {
					fmt.Fprintf(w, `{"code":200,"data":{"method":"PUT","signedUrl":%q,"headers":{}}}`, srv.URL+"/storage")
					return
				}
				io.Copy(io.Discard, r.Body)
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer srv.Close()
			client, err := duoplus.NewClient("fixture-key", duoplus.WithBaseURL(srv.URL))
			if err != nil {
				t.Fatal(err)
			}
			result, err := client.CloudDisk.Upload(context.Background(), clouddisk.SignedURLRequest{Name: "a.txt"}, strings.NewReader("x"), 1)
			var apiErr *duoplus.APIError
			if result != nil || !errors.As(err, &apiErr) || apiErr.HTTPStatus != tc.status || apiErr.Code != tc.code || apiErr.Message != tc.message {
				t.Fatalf("lost storage failure: %v", err)
			}
		})
	}
}
