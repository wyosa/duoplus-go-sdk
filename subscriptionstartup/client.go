// Package subscriptionstartup lists, purchases, and renews subscription startups.
package subscriptionstartup

import (
	"context"

	"github.com/wyosa/duoplus-go-sdk/common"
	"github.com/wyosa/duoplus-go-sdk/internal/clientcore"
)

const (
	pathList     = "/api/v1/subscriptionStartup/list"
	pathPurchase = "/api/v1/subscriptionStartup/purchase"
	pathRenewal  = "/api/v1/subscriptionStartup/renewal"

	pathAutoRenewal = "/api/v1/subscriptionStartup/autoRenewal"
)

type Client struct {
	core *clientcore.Client
}

func New(core *clientcore.Client) *Client {
	return &Client{core: core}
}

// ListRequest allows at most 100 items per page.
type ListRequest struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Remark string `json:"remark,omitempty"`
	// RenewalStatus is 0 for off or 1 for on; nil omits the filter.
	RenewalStatus *int `json:"renewal_status,omitempty"`
	// 0: unavailable; 1: available; nil: no filter (CN).
	FreeStatus *int `json:"free_status,omitempty"`
	// SortBy is name, created_at, or expired_at.
	SortBy string `json:"sort_by,omitempty"`
	// Order is asc or desc.
	Order string `json:"order,omitempty"`
	common.PaginationRequest
}

type Item struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	CPU  string `json:"cpu"`
	RAM  string `json:"ram"`
	ROM  string `json:"rom"`
	// RenewalStatus is 0 for disabled or 1 for enabled.
	RenewalStatus int `json:"renewal_status"`
	// FreeStatus is 0 for unavailable or 1 for available.
	FreeStatus int    `json:"free_status"`
	Remark     string `json:"remark"`
	// ExpiredAt and CreatedAt are Unix timestamps in seconds encoded as strings.
	ExpiredAt   string `json:"expired_at"`
	CreatedAt   string `json:"created_at"`
	NeedRenewal bool   `json:"need_renewal"`
}

type ListResponse struct {
	List []Item `json:"list"`
	common.Pagination
}

type PurchaseRequest struct {
	// Duration in days; defaults to 30 when omitted.
	Duration   string `json:"duration,omitempty"`
	Quantity   int    `json:"quantity"`
	CouponCode string `json:"coupon_code,omitempty"`
	// RenewalStatus is 0 for off or 1 for on; nil uses the server default of 1.
	RenewalStatus *int `json:"renewal_status,omitempty"`
}

type RenewRequest struct {
	PhoneIDs []string `json:"phone_ids"`
	// Duration in days; defaults to 30 when omitted.
	Duration   string `json:"duration,omitempty"`
	CouponCode string `json:"coupon_code,omitempty"`
}

type AutoRenewalRequest struct {
	PhoneIDs []string `json:"phone_ids"`
	// Status is 0 for disabled or 1 for enabled.
	Status int `json:"status"`
}

func (c *Client) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	var resp ListResponse
	if err := c.core.Do(ctx, pathList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Purchase(ctx context.Context, req PurchaseRequest) (*common.OrderResponse, error) {
	var resp common.OrderResponse
	if err := c.core.Do(ctx, pathPurchase, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Renew(ctx context.Context, req RenewRequest) (*common.OrderResponse, error) {
	var resp common.OrderResponse
	if err := c.core.Do(ctx, pathRenewal, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SetAutoRenewal(ctx context.Context, phoneIDs []string, enabled bool) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathAutoRenewal, AutoRenewalRequest{PhoneIDs: phoneIDs, Status: common.BoolToInt(enabled)}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
