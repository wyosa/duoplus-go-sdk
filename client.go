// Package duoplus is a Go client for the DuoPlus cloud phone OpenAPI.
package duoplus

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/wyosa/duoplus-go-sdk/app"
	"github.com/wyosa/duoplus-go-sdk/automation"
	"github.com/wyosa/duoplus-go-sdk/clouddisk"
	"github.com/wyosa/duoplus-go-sdk/cloudnumber"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
	"github.com/wyosa/duoplus-go-sdk/group"
	"github.com/wyosa/duoplus-go-sdk/internal/clientcore"
	"github.com/wyosa/duoplus-go-sdk/proxy"
	"github.com/wyosa/duoplus-go-sdk/subscriptionstartup"
	"github.com/wyosa/duoplus-go-sdk/team"
)

const (
	DefaultBaseURL     = clientcore.DefaultBaseURL
	DefaultIntlBaseURL = clientcore.DefaultIntlBaseURL
	DefaultCNBaseURL   = clientcore.DefaultCNBaseURL
	DefaultLanguage    = clientcore.DefaultLanguage
)

// APIError reports HTTP or API failures; inspect it with errors.As.
type APIError = clientcore.APIError

type config struct {
	baseURL     string
	lang        string
	httpClient  *http.Client
	minInterval time.Duration
}

type Option func(*config)

// Client shares one HTTP client and per-endpoint limiter across services.
type Client struct {
	CloudPhones          *cloudphone.Client
	CloudNumbers         *cloudnumber.Client
	Groups               *group.Client
	Proxies              *proxy.Client
	SubscriptionStartups *subscriptionstartup.Client
	Apps                 *app.Client
	CloudDisk            *clouddisk.Client
	Automation           *automation.Client
	Team                 *team.Client

	core *clientcore.Client
}

// NewClient defaults to the international host, English, a 30s HTTP timeout,
// and a 1s interval per endpoint. Reuse the client; only rate-limited requests
// (HTTP or API code 429) are retried, at most twice.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("duoplus api key is required")
	}

	cfg := config{
		baseURL:     DefaultBaseURL,
		lang:        DefaultLanguage,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		minInterval: time.Second,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	core, err := clientcore.New(apiKey, clientcore.Config{
		BaseURL:     cfg.baseURL,
		Lang:        cfg.lang,
		HTTPClient:  cfg.httpClient,
		MinInterval: cfg.minInterval,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		CloudPhones:          cloudphone.New(core),
		CloudNumbers:         cloudnumber.New(core),
		Groups:               group.New(core),
		Proxies:              proxy.New(core),
		SubscriptionStartups: subscriptionstartup.New(core),
		Apps:                 app.New(core),
		CloudDisk:            clouddisk.New(core),
		Automation:           automation.New(core),
		Team:                 team.New(core),
		core:                 core,
	}, nil
}

// WithBaseURL requires HTTPS, except for loopback HTTP servers.
func WithBaseURL(baseURL string) Option {
	return func(cfg *config) {
		if strings.TrimSpace(baseURL) != "" {
			cfg.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

// WithLanguage sets the Lang header: en, zh, zh-TW, or ru.
func WithLanguage(lang string) Option {
	return func(cfg *config) {
		if strings.TrimSpace(lang) != "" {
			cfg.lang = lang
		}
	}
}

// WithHTTPClient copies the client and disables redirects. Transport and Jar remain shared.
// Use a context deadline to include rate-limit waits and storage uploads.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(cfg *config) {
		if httpClient != nil {
			cfg.httpClient = httpClient
		}
	}
}

// WithMinInterval sets the per-endpoint interval for this client, counted from when the previous
// request was written; non-positive values are ignored.
func WithMinInterval(interval time.Duration) Option {
	return func(cfg *config) {
		if interval > 0 {
			cfg.minInterval = interval
		}
	}
}
