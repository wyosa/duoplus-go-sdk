package cloudnumber

import (
	"context"

	"github.com/wyosa/duoplus-go-sdk/common"
	"github.com/wyosa/duoplus-go-sdk/internal/clientcore"
)

const (
	pathList     = "/api/v1/cloudNumber/numberList"
	pathSMSList  = "/api/v1/cloudNumber/smsList"
	pathWriteSMS = "/api/v1/cloudNumber/imageWriteSms"

	pathPurchasePackage = "/api/v1/cloudNumber/purchasePackage"
	pathPackage         = "/api/v1/cloudNumber/package"
	pathPurchase        = "/api/v1/cloudNumber/purchase"
	pathRenewalPackage  = "/api/v1/cloudNumber/renewalPackage"
	pathRenewal         = "/api/v1/cloudNumber/renewal"
	// The EN docs print /api/v1/proxy/autoRenewal here by mistake; the CN docs give the real path.
	pathAutoRenewal = "/api/v1/cloudNumber/autoRenewal"
)

type Client struct {
	core *clientcore.Client
}

func New(core *clientcore.Client) *Client {
	return &Client{core: core}
}

// ListRequest allows at most 100 items per page.
type ListRequest struct {
	IDs         []string `json:"id,omitempty"`
	PhoneNumber string   `json:"phone_number,omitempty"`
	// Status is 1 for active numbers or 2 for expired numbers.
	Status []int `json:"status,omitempty"`
	// TypeIDs uses 0 for VOIP and 1 for non-VOIP.
	TypeIDs   []int    `json:"type_ids,omitempty"`
	RegionIDs []string `json:"region_ids,omitempty"`
	// RenewalStatus uses 0 for disabled or 1 for enabled.
	RenewalStatus []int  `json:"renewal_status,omitempty"`
	Remark        string `json:"remark,omitempty"`
	// SortBy is created_at or expired_at.
	SortBy string `json:"sort_by,omitempty"`
	// Order is asc or desc.
	Order string `json:"order,omitempty"`
	common.PaginationRequest
}

type Number struct {
	ID            string `json:"id"`
	PhoneNumber   string `json:"phone_number"`
	RegionName    string `json:"region_name"`
	TypeName      string `json:"type_name"`
	StatusName    string `json:"status_name"`
	RenewalStatus int    `json:"renewal_status"`
	Assigned      bool   `json:"assigned"`
	Remark        string `json:"remark"`
	// CreatedAt and ExpiredAt use 2006-01-02 15:04:05 in GMT+08:00.
	CreatedAt string `json:"created_at"`
	ExpiredAt string `json:"expired_at"`
}

type ListResponse struct {
	List []Number `json:"list"`
	common.Pagination
}

// SMSListRequest allows at most 100 items per page.
type SMSListRequest struct {
	NumberID string `json:"number_id"`
	common.PaginationRequest
}

type SMS struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	// ReceivedAt uses 2006-01-02 15:04:05 in GMT+08:00.
	ReceivedAt string `json:"received_at"`
}

type SMSListResponse struct {
	List []SMS `json:"list"`
	common.Pagination
}

// WriteSMSRequest sends image_id as an array (EN); CN documents a string.
type WriteSMSRequest struct {
	ImageIDs []string  `json:"image_id"`
	SMS      []SMSItem `json:"sms"`
}

type SMSItem struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

type PackageCategory struct {
	// Type is 0 for VOIP or 1 for non-VOIP.
	Type    int              `json:"type"`
	Name    string           `json:"name"`
	Country []PackageCountry `json:"country"`
}

type PackageCountry struct {
	Region    string            `json:"region"`
	Name      string            `json:"name"`
	Durations []PackageDuration `json:"durations"`
}

type PackageDuration struct {
	Name string `json:"name"`
	// Duration in days, for PurchaseRequest.Duration.
	Days int `json:"days"`
	// Additional gifted days.
	GifDays int `json:"gif_days"`
	// Random gifted days, as a display string.
	RandomGifDays  string  `json:"random_gif_days"`
	IsSpecialOffer int     `json:"is_special_offer"`
	Price          float64 `json:"price"`
	CardPrice      float64 `json:"card_price"`
	TotalPrice     float64 `json:"total_price"`
}

type PackageDurationsRequest struct {
	Region string `json:"region"`
	// Type is 0 for VOIP or 1 for non-VOIP; nil uses the server default of 1.
	Type *int `json:"type,omitempty"`
}

// PackageDurationsResponse contains display labels; use PackageDuration.Days for purchases.
type PackageDurationsResponse struct {
	Duration []string `json:"duration"`
}

type PurchaseRequest struct {
	Region string `json:"region"`
	// Duration is in days; obtain region-specific values from PurchasePackages.
	Duration int `json:"duration"`
	// Type is 0 for VOIP or 1 for non-VOIP; nil uses the server default of 1.
	Type *int `json:"type,omitempty"`
	// Quantity defaults to 1 when zero is omitted.
	Quantity   int    `json:"quantity,omitempty"`
	CouponCode string `json:"coupon_code,omitempty"`
	// RenewalStatus is 0 for off or 1 for on; nil uses the server default of 1.
	RenewalStatus *int `json:"renewal_status,omitempty"`
}

type RenewalPackagesRequest struct {
	NumberIDs []string `json:"number_ids"`
}

type RenewalPackage struct {
	ID          string `json:"id"`
	PhoneNumber string `json:"phone_number"`
	// ExpiredAt is a Unix timestamp in seconds encoded as a string.
	ExpiredAt string `json:"expired_at"`
	// Duration contains available renewal durations in days.
	Duration []int `json:"duration"`
}

type RenewalPackagesResponse struct {
	Numbers []RenewalPackage `json:"numbers"`
}

type RenewRequest struct {
	List []RenewItem `json:"list"`
}

type RenewItem struct {
	NumberIDs []string `json:"number_ids"`
	// Duration is in days; obtain supported values from RenewalPackages.
	Duration int `json:"duration"`
}

type AutoRenewalRequest struct {
	NumberIDs []string `json:"number_ids"`
	// Type is 0 for disabled or 1 for enabled.
	Type int `json:"type"`
}

func (c *Client) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	var resp ListResponse
	if err := c.core.Do(ctx, pathList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SMSList(ctx context.Context, req SMSListRequest) (*SMSListResponse, error) {
	var resp SMSListResponse
	if err := c.core.Do(ctx, pathSMSList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// WriteSMS supports Android 12A, 15, and 15 Pro.
func (c *Client) WriteSMS(ctx context.Context, req WriteSMSRequest) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathWriteSMS, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) PurchasePackages(ctx context.Context) ([]PackageCategory, error) {
	var resp []PackageCategory
	if err := c.core.Do(ctx, pathPurchasePackage, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) PackageDurations(ctx context.Context, req PackageDurationsRequest) (*PackageDurationsResponse, error) {
	var resp PackageDurationsResponse
	if err := c.core.Do(ctx, pathPackage, req, &resp); err != nil {
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

func (c *Client) RenewalPackages(ctx context.Context, numberIDs []string) (*RenewalPackagesResponse, error) {
	var resp RenewalPackagesResponse
	if err := c.core.Do(ctx, pathRenewalPackage, RenewalPackagesRequest{NumberIDs: numberIDs}, &resp); err != nil {
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

func (c *Client) SetAutoRenewal(ctx context.Context, numberIDs []string, enabled bool) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathAutoRenewal, AutoRenewalRequest{NumberIDs: numberIDs, Type: common.BoolToInt(enabled)}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
