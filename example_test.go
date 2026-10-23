package duoplus_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
	"github.com/wyosa/duoplus-go-sdk/common"
)

func ExampleNewClient() {
	client, err := duoplus.NewClient("your-api-key",
		duoplus.WithBaseURL(duoplus.DefaultCNBaseURL),
		duoplus.WithLanguage("zh"),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(client.CloudPhones != nil, client.Team != nil)
	// Output: true true
}

func ExampleClient_pagination() {
	// A local server keeps this example executable without API credentials.
	client, closeServer := exampleClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req cloudphone.ListRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			panic(err)
		}
		fmt.Fprintf(w, `{"code":200,"data":{"list":[{"id":"phone-%d","status":1}],"page":%d,"pagesize":1,"total":2,"total_page":2}}`, req.Page, req.Page)
	}))
	defer closeServer()

	// This deadline covers both rate-limit waits and HTTP requests.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for page := 1; ; page++ {
		result, err := client.CloudPhones.List(ctx, cloudphone.ListRequest{
			PaginationRequest: common.PaginationRequest{Page: page, PageSize: 1},
		})
		if err != nil {
			panic(err)
		}
		for _, phone := range result.List {
			fmt.Println(phone.ID, phone.Status)
		}
		if len(result.List) == 0 || page >= result.TotalPage {
			break
		}
	}
	// Output:
	// phone-1 1
	// phone-2 1
}

func ExampleClient_partialFailure() {
	client, closeServer := exampleClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"code":200,"data":{"success":["phone-1"],"fail":["phone-2"],"fail_reason":{"phone-2":"expired"}}}`)
	}))
	defer closeServer()

	result, err := client.CloudPhones.PowerOn(context.Background(), []string{"phone-1", "phone-2"})
	if err != nil {
		panic(err)
	}
	for _, id := range result.Fail {
		fmt.Println(id, result.FailReason[id])
	}
	// Accepted phones still need Status polling before they are ready.
	fmt.Println("accepted:", len(result.Success))
	// Output:
	// phone-2 expired
	// accepted: 1
}

func ExampleAPIError() {
	client, closeServer := exampleClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"code":429,"message":"quota exceeded"}`)
	}))
	defer closeServer()

	_, err := client.CloudPhones.List(context.Background(), cloudphone.ListRequest{})
	var apiErr *duoplus.APIError
	if errors.As(err, &apiErr) {
		// The HTTP status and envelope code are independent.
		fmt.Println(apiErr.HTTPStatus, apiErr.Code)
	}
	// Output: 429 429
}

func ExampleClient_resources() {
	client, closeServer := exampleClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"code":200,"data":{"balance":"9007199254740993.01","phone_count":2}}`)
	}))
	defer closeServer()

	resources, err := client.Team.Resources(context.Background())
	if err != nil {
		panic(err)
	}
	// String preserves the decimal value; Float64 may lose precision.
	fmt.Println(resources.Balance.String())
	// Output: 9007199254740993.01
}

func exampleClient(handler http.Handler) (*duoplus.Client, func()) {
	server := httptest.NewServer(handler)
	client, err := duoplus.NewClient("fixture-key",
		duoplus.WithBaseURL(server.URL),
		// Only local fixtures use a shorter interval than the API's 1 QPS limit.
		duoplus.WithMinInterval(time.Nanosecond),
	)
	if err != nil {
		server.Close()
		panic(err)
	}
	return client, server.Close
}
