// Package proxy manages, checks, and renews proxies.
package proxy

import (
	"context"

	"github.com/wyosa/duoplus-go-sdk/common"
	"github.com/wyosa/duoplus-go-sdk/internal/clientcore"
)

const (
	pathList    = "/api/v1/proxy/list"
	pathAdd     = "/api/v1/proxy/add"
	pathDelete  = "/api/v1/proxy/delete"
	pathRefresh = "/api/v1/proxy/refresh"
	pathUpdate  = "/api/v1/proxy/update"

	pathCheck       = "/api/v1/proxy/check"
	pathAutoRenewal = "/api/v1/proxy/autoRenewal"
)

type Client struct {
	core *clientcore.Client
}

func New(core *clientcore.Client) *Client {
	return &Client{core: core}
}

// ListRequest allows at most 100 items per page.
type ListRequest struct {
	// 1: connected; 2: connection failed.
	Status []int `json:"status,omitempty"`
	common.PaginationRequest
}

type Item struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Host      string   `json:"host"`
	Port      string   `json:"port"`
	User      string   `json:"user"`
	Area      string   `json:"area"`
	GroupIDs  []string `json:"group_ids"`
	GroupName []string `json:"group_name"`
}

type ListResponse struct {
	List []Item `json:"list"`
	common.Pagination
}

// AddRequest accepts at most 20 proxies.
type AddRequest struct {
	ProxyList []Input `json:"proxy_list"`
	// IPScanChannel defaults to ip2location when omitted.
	IPScanChannel string `json:"ip_scan_channel,omitempty"`
}

// Input does not support IPv6 proxies.
type Input struct {
	// Protocol is socks5, http, or https.
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name,omitempty"`
}

type AddSuccess struct {
	Index int    `json:"index"`
	ID    string `json:"id"`
}

type AddFailure struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

// AddResponse can contain Fail entries even when err is nil.
type AddResponse struct {
	Success []AddSuccess `json:"success"`
	Fail    []AddFailure `json:"fail"`
}

type IDsRequest struct {
	IDs []string `json:"ids"`
}

// CheckRequest accepts a ProxyID or Host and Port.
type CheckRequest struct {
	ProxyID string `json:"proxy_id,omitempty"`
	// Protocol is socks5, http, or https; omission defaults to socks5.
	Protocol string `json:"protocol,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	// IPScanChannel defaults to ip2location when omitted.
	IPScanChannel string `json:"ip_scan_channel,omitempty"`
}

// CheckResponse.Success can be false even when err is nil; see Tips.
type CheckResponse struct {
	Success   bool   `json:"success"`
	IP        string `json:"ip"`
	Country   string `json:"country"`
	Region    string `json:"region"`
	City      string `json:"city"`
	Timezone  string `json:"timezone"`
	Longitude string `json:"longitude"`
	Latitude  string `json:"latitude"`
	Tips      string `json:"tips"`
	User      string `json:"user"`
	Password  string `json:"password"`
}

type AutoRenewalRequest struct {
	ProxyIDs []string `json:"proxy_ids"`
	// Type is 0 for disabled or 1 for enabled.
	Type int `json:"type"`
}

// UpdateRequest omits nil fields; common.Ptr("") clears a string. The API documents socks5 and
// ip2location for an omitted Protocol and IPScanChannel; set both when the proxy uses other values.
type UpdateRequest struct {
	ID string `json:"id"`
	// Protocol is socks5, http, or https; the API documents socks5 when it is omitted.
	Protocol *string `json:"protocol,omitempty"`
	Host     *string `json:"host,omitempty"`
	Port     int     `json:"port,omitempty"`
	User     *string `json:"user,omitempty"`
	Password *string `json:"password,omitempty"`
	Name     *string `json:"name,omitempty"`
	// IPScanChannel is ip2location or ipapi; the API documents ip2location when it is omitted.
	IPScanChannel *string `json:"ip_scan_channel,omitempty"`
	RefreshURL    *string `json:"refresh_url,omitempty"`
	ProxyURL      *string `json:"proxy_url,omitempty"`
}

type UpdateResponse struct {
	Message string `json:"message"`
	// Result is undocumented; the published example is an empty array.
	Result any `json:"result"`
}

func (c *Client) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	var resp ListResponse
	if err := c.core.Do(ctx, pathList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Add(ctx context.Context, req AddRequest) (*AddResponse, error) {
	var resp AddResponse
	if err := c.core.Do(ctx, pathAdd, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Delete(ctx context.Context, ids []string) (*common.IDsOperationResponse, error) {
	var resp common.IDsOperationResponse
	if err := c.core.Do(ctx, pathDelete, IDsRequest{IDs: ids}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) RefreshURLs(ctx context.Context, ids []string) (*common.IDsOperationResponse, error) {
	var resp common.IDsOperationResponse
	if err := c.core.Do(ctx, pathRefresh, IDsRequest{IDs: ids}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Update(ctx context.Context, req UpdateRequest) (*UpdateResponse, error) {
	var resp UpdateResponse
	if err := c.core.Do(ctx, pathUpdate, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Check(ctx context.Context, req CheckRequest) (*CheckResponse, error) {
	var resp CheckResponse
	if err := c.core.Do(ctx, pathCheck, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SetAutoRenewal(ctx context.Context, proxyIDs []string, enabled bool) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathAutoRenewal, AutoRenewalRequest{ProxyIDs: proxyIDs, Type: common.BoolToInt(enabled)}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
