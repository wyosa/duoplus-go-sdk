package duoplus

import (
	"encoding/json"
	"testing"

	"github.com/wyosa/duoplus-go-sdk/common"
)

func TestTextUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		data string
		want common.Text
	}{
		{name: "string", data: `"abc"`, want: common.Text("abc")},
		{name: "number", data: `12345`, want: common.Text("12345")},
		{name: "bool", data: `true`, want: common.Text("true")},
		{name: "null", data: `null`, want: common.Text("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got common.Text
			if err := json.Unmarshal([]byte(tt.data), &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Unmarshal() got = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewClientRequiresAPIKey(t *testing.T) {
	if _, err := NewClient(""); err == nil {
		t.Fatal("expected error for empty api key")
	}
}

func TestNewClientRejectsInsecureBaseURL(t *testing.T) {
	if _, err := NewClient("test-key", WithBaseURL("http://openapi.duoplus.net")); err == nil {
		t.Fatal("expected error for plain http base url")
	}
}

func TestNewClientInitializesServices(t *testing.T) {
	client, err := NewClient("test-key")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.CloudPhones == nil || client.CloudNumbers == nil || client.Groups == nil {
		t.Fatal("expected core service fields to be initialized")
	}

	if client.Proxies == nil || client.SubscriptionStartups == nil || client.Apps == nil {
		t.Fatal("expected module service fields to be initialized")
	}

	if client.CloudDisk == nil || client.Automation == nil || client.Team == nil {
		t.Fatal("expected auxiliary service fields to be initialized")
	}
}
