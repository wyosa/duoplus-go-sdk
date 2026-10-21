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

type APIError = clientcore.APIError

type config struct {
	baseURL     string
	lang        string
	httpClient  *http.Client
	minInterval time.Duration
}

type Option func(*config)

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

func WithBaseURL(baseURL string) Option {
	return func(cfg *config) {
		if strings.TrimSpace(baseURL) != "" {
			cfg.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

func WithLanguage(lang string) Option {
	return func(cfg *config) {
		if strings.TrimSpace(lang) != "" {
			cfg.lang = lang
		}
	}
}

// WithHTTPClient sets the HTTP client used for API calls. The client is copied
// and redirects are always refused so the API key never leaves the API host.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(cfg *config) {
		if httpClient != nil {
			cfg.httpClient = httpClient
		}
	}
}

func WithMinInterval(interval time.Duration) Option {
	return func(cfg *config) {
		if interval > 0 {
			cfg.minInterval = interval
		}
	}
}
