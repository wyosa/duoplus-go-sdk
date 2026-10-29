// Package team reads team orders, balance, and resources.
package team

import (
	"context"

	"github.com/wyosa/duoplus-go-sdk/common"
	"github.com/wyosa/duoplus-go-sdk/internal/clientcore"
)

const (
	pathOrder    = "/api/v1/team/order"
	pathBalance  = "/api/v1/team/balance"
	pathResource = "/api/v1/team/resource"
)

type Client struct {
	core *clientcore.Client
}

func New(core *clientcore.Client) *Client {
	return &Client{core: core}
}

// OrderListRequest allows at most 500 items per page.
type OrderListRequest struct {
	// CreatedAtStart and CreatedAtEnd: RFC 3339 with offset, e.g. 2026-01-01T00:00:00+08:00.
	// Omitting both selects the last seven days in GMT+08:00.
	CreatedAtStart string   `json:"created_at_start,omitempty"`
	CreatedAtEnd   string   `json:"created_at_end,omitempty"`
	ImageIDs       []string `json:"image_ids,omitempty"`
	OrderID        string   `json:"order_id,omitempty"`
	// Status contains decimal strings: "1" (unpaid), "2" (canceled), "3" (paid), "4" (failed), "5"
	// (finished), "6" (expired), or "7" (refunded).
	Status []string `json:"status,omitempty"`
	// Type uses decimal strings: 1 purchase, 2 renewal, 3 recharge, 4 deduction, 5 redeem, 6 gift,
	// 7 refund, 8 transfer, or 9 receive.
	Type        []string `json:"type,omitempty"`
	ProductType []string `json:"product_type,omitempty"`
	// SortBy is created_at or total.
	SortBy string `json:"sort_by,omitempty"`
	// Order is asc or desc.
	Order string `json:"order,omitempty"`
	common.PaginationRequest
}

type Order struct {
	Type        string `json:"type"`
	OrderID     string `json:"order_id"`
	Product     string `json:"product"`
	Description string `json:"description"`
	Status      string `json:"status"`
	// Formatted currency amount, including its sign.
	Total          string `json:"total"`
	CreatedAt      string `json:"created_at"`
	ExpiredAt      string `json:"expired_at"`
	ExpiredSeconds int    `json:"expired_seconds"`
}

type OrderListResponse struct {
	List []Order `json:"list"`
	common.Pagination
}

// Balance contains currency-formatted strings.
type Balance struct {
	Balance         string `json:"balance"`
	RechargeBalance string `json:"recharge_balance"`
	GiftBalance     string `json:"gift_balance"`
}

type balanceResponse struct {
	Balance Balance `json:"balance"`
}

type Resources struct {
	// Balance keeps the exact decimal text of a JSON number or string.
	Balance          common.Text `json:"balance"`
	PhoneCount       int         `json:"phone_count"`
	ImageCount       int         `json:"image_count"`
	PaidNewMockTimes int         `json:"paid_new_mock_times"`
}

func (c *Client) Orders(ctx context.Context, req OrderListRequest) (*OrderListResponse, error) {
	var resp OrderListResponse
	if err := c.core.Do(ctx, pathOrder, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Balance contains currency-formatted strings.
func (c *Client) Balance(ctx context.Context) (*Balance, error) {
	var resp balanceResponse
	if err := c.core.Do(ctx, pathBalance, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Balance, nil
}

func (c *Client) Resources(ctx context.Context) (*Resources, error) {
	var resp Resources
	if err := c.core.Do(ctx, pathResource, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
