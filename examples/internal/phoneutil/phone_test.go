package phoneutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
)

func testClient(t *testing.T, handler http.HandlerFunc) *duoplus.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := duoplus.NewClient("test-key", duoplus.WithBaseURL(srv.URL), duoplus.WithMinInterval(time.Nanosecond))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSelectFiltersExplicitID(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req cloudphone.ListRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if reflect.DeepEqual(req.ImageIDs, []string{"page-2-phone"}) {
			fmt.Fprint(w, `{"code":200,"data":{"list":[{"id":"page-2-phone","status":2}]}}`)
			return
		}
		fmt.Fprint(w, `{"code":200,"data":{"list":[{"id":"first-page-phone","status":1}],"total":21,"total_page":2}}`)
	})
	phone, err := Select(context.Background(), c, "page-2-phone")
	if err != nil || phone.ID != "page-2-phone" {
		t.Fatalf("targeted lookup failed: %+v, %v", phone, err)
	}
}

func TestSelectRejectsMissingExplicitID(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"code":200,"data":{"list":[{"id":"other","status":1}]}}`)
	})
	if _, err := Select(context.Background(), c, "missing"); err == nil {
		t.Fatal("selected a different phone")
	}
}

func TestEnsureOnWaitsForStartup(t *testing.T) {
	for _, initial := range []int{statusPoweredOff, statusPoweringOn, statusConfiguring} {
		t.Run(fmt.Sprint(initial), func(t *testing.T) {
			var powerCalls, statusCalls atomic.Int32
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/cloudPhone/powerOn":
					powerCalls.Add(1)
					fmt.Fprint(w, `{"code":200,"data":{"success":["p1"],"fail":[]}}`)
				case "/api/v1/cloudPhone/status":
					status := statusPoweringOn
					if statusCalls.Add(1) > 1 {
						status = statusPoweredOn
					}
					fmt.Fprintf(w, `{"code":200,"data":{"list":[{"id":"p1","status":%d}]}}`, status)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
				}
			})
			err := EnsureOn(context.Background(), c, &cloudphone.Phone{ID: "p1", Status: initial}, time.Second, time.Millisecond)
			if err != nil || statusCalls.Load() != 2 {
				t.Fatalf("did not wait for readiness: polls=%d, err=%v", statusCalls.Load(), err)
			}
			wantPower := int32(0)
			if initial == statusPoweredOff {
				wantPower = 1
			}
			if powerCalls.Load() != wantPower {
				t.Fatalf("power calls=%d, want %d", powerCalls.Load(), wantPower)
			}
		})
	}
}

func TestEnsureOnRejectsFailedPowerOn(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/cloudPhone/powerOn" {
			t.Error("polled after power-on failure")
		}
		fmt.Fprint(w, `{"code":200,"data":{"success":[],"fail":["p1"],"fail_reason":{"p1":"no resources"}}}`)
	})
	if err := EnsureOn(context.Background(), c, &cloudphone.Phone{ID: "p1", Status: 2}, time.Second, time.Millisecond); err == nil {
		t.Fatal("accepted a failed power-on")
	}
}

func TestEnsureOnDeadlineCancelsInFlightRequests(t *testing.T) {
	for _, initial := range []int{statusPoweredOff, statusPoweringOn} {
		t.Run(fmt.Sprint(initial), func(t *testing.T) {
			canceled := make(chan struct{}, 1)
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				// Drain the request so HTTP/1 can observe the client closing
				// the connection while the handler waits for cancellation.
				io.Copy(io.Discard, r.Body)
				select {
				case <-r.Context().Done():
					canceled <- struct{}{}
				case <-time.After(time.Second):
					fmt.Fprint(w, `{"code":200,"data":{"success":["p1"],"list":[{"id":"p1","status":1}]}}`)
				}
			})
			err := EnsureOn(context.Background(), c, &cloudphone.Phone{ID: "p1", Status: initial}, 50*time.Millisecond, time.Millisecond)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("deadline not enforced: %v", err)
			}
			select {
			case <-canceled:
			case <-time.After(time.Second):
				t.Fatal("HTTP request outlived the polling deadline")
			}
		})
	}
}

func TestEnsureOnCancellationInterruptsPollingSleep(t *testing.T) {
	polled := make(chan struct{}, 1)
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"code":200,"data":{"list":[{"id":"p1","status":10}]}}`)
		polled <- struct{}{}
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		finished <- EnsureOn(ctx, c, &cloudphone.Phone{ID: "p1", Status: 10}, 2*time.Second, time.Second)
	}()
	<-polled
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("cancellation waited for the polling interval")
	}
}

func TestEnsureOnRejectsMissingOrUnusableStatus(t *testing.T) {
	for _, list := range []string{`[]`, `[{"id":"other","status":1}]`, `[{"id":"p1","status":12}]`, `[{"id":"p1","status":3}]`} {
		t.Run(list, func(t *testing.T) {
			c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintf(w, `{"code":200,"data":{"list":%s}}`, list)
			})
			if err := EnsureOn(context.Background(), c, &cloudphone.Phone{ID: "p1", Status: 10}, time.Second, time.Millisecond); err == nil {
				t.Fatal("reported an unavailable phone as ready")
			}
		})
	}
}
