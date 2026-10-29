package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	duoplus "github.com/wyosa/duoplus-go-sdk"
)

func TestWorkflowChecksReadinessAndPushResult(t *testing.T) {
	for _, tc := range []struct {
		name    string
		push    string
		wantErr bool
	}{
		{"success", `{"success":[{"image_id":"p1","id":"f1"}],"fail":[]}`, false},
		{"failed push", `{"success":[],"fail":[{"image_id":"p1","id":"f1","err":"offline"}]}`, true},
		{"unconfirmed push", `{"success":[],"fail":[]}`, true},
		{"wrong destination", `{"success":[{"image_id":"other","id":"f1"}],"fail":[]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DUOPLUS_EXECUTE", "1")
			t.Setenv("DUOPLUS_TARGET_IMAGE_ID", "")
			var statusChecked, pushed atomic.Bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data := `{}`
				switch r.URL.Path {
				case "/api/v1/cloudPhone/list":
					data = `{"list":[{"id":"p1","status":10}]}`
				case "/api/v1/cloudDisk/list":
					data = `{"list":[{"id":"f1"}]}`
				case "/api/v1/cloudPhone/status":
					statusChecked.Store(true)
					data = `{"list":[{"id":"p1","status":1}]}`
				case "/api/v1/cloudDisk/pushFiles":
					if !statusChecked.Load() {
						t.Error("pushed before checking the starting phone's readiness")
					}
					pushed.Store(true)
					data = tc.push
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
				}
				fmt.Fprintf(w, `{"code":200,"data":%s}`, data)
			}))
			defer srv.Close()
			client, err := duoplus.NewClient("test-key", duoplus.WithBaseURL(srv.URL), duoplus.WithMinInterval(time.Nanosecond))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err = run(ctx, client)
			if (err != nil) != tc.wantErr || !pushed.Load() {
				t.Fatalf("pushed=%v, err=%v, wantErr=%v", pushed.Load(), err, tc.wantErr)
			}
		})
	}
}

func TestWorkflowDryRunDoesNotChangeResources(t *testing.T) {
	t.Setenv("DUOPLUS_EXECUTE", "")
	t.Setenv("DUOPLUS_TARGET_IMAGE_ID", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/cloudPhone/list":
			fmt.Fprint(w, `{"code":200,"data":{"list":[{"id":"p1","status":2}]}}`)
		case "/api/v1/cloudDisk/list":
			fmt.Fprint(w, `{"code":200,"data":{"list":[{"id":"f1"}]}}`)
		default:
			t.Errorf("dry run made unexpected call %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer srv.Close()
	client, err := duoplus.NewClient("test-key", duoplus.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), client); err != nil {
		t.Fatal(err)
	}
}
