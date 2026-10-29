// Package cloudphone manages cloud phone lifecycle, configuration, ADB, sharing, and billing.
package cloudphone

import (
	"context"

	"github.com/wyosa/duoplus-go-sdk/common"
	"github.com/wyosa/duoplus-go-sdk/internal/clientcore"
)

const (
	pathList            = "/api/v1/cloudPhone/list"
	pathPowerOn         = "/api/v1/cloudPhone/powerOn"
	pathPowerOff        = "/api/v1/cloudPhone/powerOff"
	pathRestart         = "/api/v1/cloudPhone/restart"
	pathStatus          = "/api/v1/cloudPhone/status"
	pathInfo            = "/api/v1/cloudPhone/info"
	pathUpdate          = "/api/v1/cloudPhone/update"
	pathModelList       = "/api/v1/mobile/modelList"
	pathBatchRoot       = "/api/v1/cloudPhone/batchRoot"
	pathCommand         = "/api/v1/cloudPhone/command"
	pathOpenADB         = "/api/v1/cloudPhone/openAdb"
	pathCloseADB        = "/api/v1/cloudPhone/closeAdb"
	pathSetADBWhitelist = "/api/v1/cloudPhone/setAdbIpWhitelist"
	pathUpdateSharePass = "/api/v1/cloudPhone/updateSharePassword"
	pathLinkUserList    = "/api/v1/cloudPhone/linkUserList"
	pathTagList         = "/api/v1/cloudPhone/tagList"
	pathResourceList    = "/api/v1/cloudPhone/cloudPhone"
	pathResolutionList  = "/api/v1/cloudPhone/resolutionList"
	pathPurchase        = "/api/v1/cloudPhone/purchase"
	pathRenewal         = "/api/v1/cloudPhone/renewal"
	pathShare           = "/api/v1/cloudPhone/share"
	pathInitProxy       = "/api/v1/cloudPhone/initProxy"
	pathProxySwitch     = "/api/v1/cloudPhone/proxySwitch"
	pathTimezoneList    = "/api/v1/mobile/timezoneList"
	pathLanguageList    = "/api/v1/mobile/languageList"
	pathNewPhone        = "/api/v1/cloudPhone/newPhone"
	pathLive            = "/api/v1/cloudPhone/live"
	pathScan            = "/api/v1/cloudPhone/scan"
	pathAutoRenewal     = "/api/v1/cloudPhone/autoRenewal"
	pathConvertVersion  = "/api/v1/cloudPhone/convertVersion"
)

type Client struct {
	core *clientcore.Client
}

func New(core *clientcore.Client) *Client {
	return &Client{core: core}
}

// ListRequest allows at most 100 items per page; status filters use strings.
type ListRequest struct {
	ImageIDs []string `json:"image_id,omitempty"`
	Name     string   `json:"name,omitempty"`
	GroupID  string   `json:"group_id,omitempty"`
	Remark   string   `json:"remark,omitempty"`
	IPs      []string `json:"ips,omitempty"`
	// LinkStatus contains the decimal-string forms of Phone.Status values.
	LinkStatus []string `json:"link_status,omitempty"`
	ProxyID    string   `json:"proxy_id,omitempty"`
	// ShareStatus uses "0" (unconfigured), "1" (shared), or "2" (not shared).
	ShareStatus []string `json:"share_status,omitempty"`
	// StartPhoneType uses "1" (prefer subscription), "2" (subscription), or "3" (temporary).
	StartPhoneType []string `json:"start_phone_type,omitempty"`
	// ADBStatus uses "0" (disabled) or "1" (enabled).
	ADBStatus []string `json:"adb_status,omitempty"`
	// RenewalStatus uses "0" (disabled) or "1" (enabled).
	RenewalStatus []string `json:"renewal_status,omitempty"`
	// SortBy is name, created_at, expired_at, or os.
	SortBy string `json:"sort_by,omitempty"`
	// Order is asc or desc.
	Order     string   `json:"order,omitempty"`
	UserIDs   []string `json:"user_ids,omitempty"`
	TagIDs    []string `json:"tag_ids,omitempty"`
	RegionIDs []string `json:"region_id,omitempty"`
	common.PaginationRequest
}

type GroupRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Phone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Status is 0 (unconfigured), 1 (on), 2 (off), 3 (expired), 4 (renewal overdue), 10 (powering
	// on), 11 (configuring), or 12 (configuration failed).
	Status      int    `json:"status"`
	ADB         string `json:"adb"`
	ADBPassword string `json:"adb_password"`
	// ADBStatus is 0 for disabled or 1 for enabled.
	ADBStatus int `json:"adb_status"`
	// HTTPStatus is "0" for disabled or "1" for enabled; numeric JSON is accepted too.
	HTTPStatus common.Text `json:"http_status"`
	Group      []GroupRef  `json:"group"`
	Remark     string      `json:"remark"`
	CreatedAt  string      `json:"created_at"`
	ExpiredAt  string      `json:"expired_at"`
	IP         string      `json:"ip"`
	Area       string      `json:"area"`
	Region     string      `json:"region"`
	// Display name, e.g. Android 12; not a purchase OS code.
	OS                       string `json:"os"`
	Size                     string `json:"size"`
	AndroidVersionConfigured bool   `json:"android_version_configured"`
	ConvertVersionTimes      int    `json:"convert_version_times"`
	FreeNewMockTimes         int    `json:"free_new_mock_times"`
	PaidNewMockTimes         int    `json:"paid_new_mock_times"`
	NewMockTimes             int    `json:"new_mock_times"`
}

type ListResponse struct {
	List []Phone `json:"list"`
	common.Pagination
}

type StatusRequest struct {
	ImageIDs []string `json:"image_ids"`
}

type StatusItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Status uses the same lifecycle codes as Phone.Status.
	Status int `json:"status"`
}

type StatusResponse struct {
	List []StatusItem `json:"list"`
}

// InfoRequest uses image_id from both request examples, despite the EN table's image_ids.
type InfoRequest struct {
	ImageID string `json:"image_id"`
}

type ProxySettings struct {
	ID string `json:"id,omitempty"`
	// DNS is 1 for enabled or 2 for disabled.
	DNS     int    `json:"dns,omitempty"`
	IP      string `json:"ip,omitempty"`
	Country string `json:"country,omitempty"`
	Region  string `json:"region,omitempty"`
	City    string `json:"city,omitempty"`
	Zipcode string `json:"zipcode,omitempty"`
}

type GPSSettings struct {
	// Type is 1 for simulation based on proxy IP or 2 for custom coordinates.
	Type      int     `json:"type,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
}

type LocaleSettings struct {
	// Type is 1 for simulation based on proxy IP or 2 for custom values.
	Type     int    `json:"type,omitempty"`
	Timezone string `json:"timezone,omitempty"`
	Language string `json:"language,omitempty"`
}

type SIMSettings struct {
	// Status is 1 for enabled or 2 for disabled.
	Status   int    `json:"status,omitempty"`
	Country  string `json:"country,omitempty"`
	MSISDN   string `json:"msisdn,omitempty"`
	Operator string `json:"operator,omitempty"`
	MCC      string `json:"mcc,omitempty"`
	MNC      string `json:"mnc,omitempty"`
	MSIN     string `json:"msin,omitempty"`
	ICCID    string `json:"iccid,omitempty"`
	IMSI     string `json:"imsi,omitempty"`
}

type BluetoothSettings struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
}

type WiFiSettings struct {
	// Status is 1 for enabled or 2 for disabled.
	Status int    `json:"status,omitempty"`
	Name   string `json:"name,omitempty"`
	MAC    string `json:"mac,omitempty"`
	BSSID  string `json:"bssid,omitempty"`
}

type DeviceSettings struct {
	Manufacturer string      `json:"manufacturer,omitempty"`
	Brand        string      `json:"brand,omitempty"`
	Model        string      `json:"model,omitempty"`
	IMEI         common.Text `json:"imei,omitempty"`
	SerialNo     string      `json:"serialno,omitempty"`
	AndroidID    string      `json:"android_id,omitempty"`
	Name         string      `json:"name,omitempty"`
	GSFID        string      `json:"gsf_id,omitempty"`
	GAID         string      `json:"gaid,omitempty"`
}

type Detail struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Remark      string            `json:"remark"`
	OS          string            `json:"os"`
	ADB         string            `json:"adb"`
	ADBPassword string            `json:"adb_password"`
	ADBStatus   int               `json:"adb_status"`
	Group       []GroupRef        `json:"group"`
	Proxy       ProxySettings     `json:"proxy"`
	GPS         GPSSettings       `json:"gps"`
	Locale      LocaleSettings    `json:"locale"`
	SIM         SIMSettings       `json:"sim"`
	Bluetooth   BluetoothSettings `json:"bluetooth"`
	WiFi        WiFiSettings      `json:"wifi"`
	Device      DeviceSettings    `json:"device"`
}

// UpdateRequest accepts at most 20 phones.
type UpdateRequest struct {
	Images []UpdateItem `json:"images"`
}

// UpdateItem leaves nil fields unchanged; common.Ptr("") sends an empty string.
type UpdateItem struct {
	ImageID   string           `json:"image_id"`
	Name      *string          `json:"name,omitempty"`
	DPIName   *string          `json:"dpi_name,omitempty"`
	Remark    *string          `json:"remark,omitempty"`
	Proxy     *ProxyUpdate     `json:"proxy,omitempty"`
	GPS       *GPSUpdate       `json:"gps,omitempty"`
	Locale    *LocaleUpdate    `json:"locale,omitempty"`
	SIM       *SIMUpdate       `json:"sim,omitempty"`
	Bluetooth *BluetoothUpdate `json:"bluetooth,omitempty"`
	WiFi      *WiFiUpdate      `json:"wifi,omitempty"`
	Device    *DeviceUpdate    `json:"device,omitempty"`
	Station   *StationUpdate   `json:"station,omitempty"`
}

type ProxyUpdate struct {
	ID *string `json:"id,omitempty"`
	// DNS is 1 for enabled or 2 for disabled; zero is omitted.
	DNS int `json:"dns,omitempty"`
}

type GPSUpdate struct {
	// 1: proxy IP; 2: custom. Required, including when coordinates are zero.
	Type      int      `json:"type"`
	Longitude *float64 `json:"longitude,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
}

type LocaleUpdate struct {
	// Type is 1 for simulation based on proxy IP or 2 for custom values; zero is omitted.
	Type int `json:"type,omitempty"`
	// Timezone is an identifier from Client.Timezones; ignored for Type 1.
	Timezone *string `json:"timezone,omitempty"`
	// Language is an identifier from Client.Languages; ignored for Type 1.
	Language *string `json:"language,omitempty"`
}

type SIMUpdate struct {
	// Status is 1 for enabled or 2 for disabled; zero is omitted.
	Status   int     `json:"status,omitempty"`
	NumberID *string `json:"number_id,omitempty"`
	Country  *string `json:"country,omitempty"`
	MSISDN   *string `json:"msisdn,omitempty"`
	Operator *string `json:"operator,omitempty"`
	MCC      *string `json:"mcc,omitempty"`
	MNC      *string `json:"mnc,omitempty"`
	MSIN     *string `json:"msin,omitempty"`
	ICCID    *string `json:"iccid,omitempty"`
	IMSI     *string `json:"imsi,omitempty"`
}

type BluetoothUpdate struct {
	Name    *string `json:"name,omitempty"`
	Address *string `json:"address,omitempty"`
}

type WiFiUpdate struct {
	// Status is 1 for enabled or 2 for disabled; zero is omitted.
	Status int     `json:"status,omitempty"`
	Name   *string `json:"name,omitempty"`
	MAC    *string `json:"mac,omitempty"`
	BSSID  *string `json:"bssid,omitempty"`
}

type DeviceUpdate struct {
	IMEI      *string `json:"imei,omitempty"`
	SerialNo  *string `json:"serialno,omitempty"`
	AndroidID *string `json:"android_id,omitempty"`
	Name      *string `json:"name,omitempty"`
	GSFID     *string `json:"gsf_id,omitempty"`
	GAID      *string `json:"gaid,omitempty"`
}

type StationUpdate struct {
	LAC *int `json:"lac,omitempty"`
	CID *int `json:"cid,omitempty"`
}

type ModelListRequest struct {
	// OS is 1 (Android 10), 2 (12A), 3 (11), 4 (15), 5 (12B), or 10 (15 Pro).
	OS int `json:"os"`
}

type ModelInfo struct {
	Name string `json:"name"`
}

type ModelListResponse map[string]map[string]ModelInfo

// BatchRootRequest accepts at most 20 phones.
type BatchRootRequest struct {
	ImageIDs []string `json:"image_ids"`
	// Status is 1 (enable all), 2 (disable all), 3 (enable packages), or 4 (disable packages).
	Status int `json:"status"`
	// Required for Status 3 or 4. Sent as packages from both request examples, despite the EN table's pkgs.
	Packages []string `json:"packages,omitempty"`
}

type CommandRequest struct {
	ImageIDs []string `json:"image_ids,omitempty"`
	ImageID  string   `json:"image_id,omitempty"`
	Command  string   `json:"command"`
}

// CommandResult.Success can be false even when err is nil.
type CommandResult struct {
	// Success reports execution success independently of the HTTP/envelope status.
	Success bool   `json:"success"`
	Content string `json:"content"`
	Message string `json:"message"`
}

// SetADBIPWhitelistRequest allows at most 10 IP addresses in the resulting whitelist.
type SetADBIPWhitelistRequest struct {
	IPs []string `json:"ips"`
	// Append is false to replace the whitelist or true to add to it.
	Append bool `json:"append,omitempty"`
}

type UpdateSharePasswordRequest struct {
	Images []SharePasswordItem `json:"images"`
}

type SharePasswordItem struct {
	ImageID  string `json:"image_id"`
	Password string `json:"password"`
}

type LinkUser struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
}

type LinkUsersResponse struct {
	List []LinkUser `json:"list"`
}

type TagListRequest struct {
	Name string `json:"name,omitempty"`
	common.PaginationRequest
}

type Tag struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	ImageCount int    `json:"image_count"`
}

type TagListResponse struct {
	List []Tag `json:"list"`
	common.Pagination
}

type Resource struct {
	Name      string `json:"name"`
	RegionID  string `json:"region_id"`
	OS        string `json:"os"`
	Count     int    `json:"count"`
	UsedCount int    `json:"used_count"`
}

type ResourceListResponse struct {
	List []Resource `json:"list"`
	common.Pagination
}

type ResolutionListResponse struct {
	List []string `json:"list"`
}

type PurchaseRequest struct {
	// OS: "10", "12A", "11", "15", "12B", or "15pro". Models uses integer codes.
	OS       string `json:"os,omitempty"`
	Quantity int    `json:"quantity"`
	// Duration in days; defaults to 30 when omitted.
	Duration   string `json:"duration,omitempty"`
	CouponCode string `json:"coupon_code,omitempty"`
	// RenewalStatus is 0 for off or 1 for on; nil uses the server default of 1.
	RenewalStatus *int `json:"renewal_status,omitempty"`
}

type RenewRequest struct {
	ImageIDs []string `json:"image_ids"`
	// Duration in days; defaults to 30 when omitted.
	Duration   string `json:"duration,omitempty"`
	CouponCode string `json:"coupon_code,omitempty"`
}

type ShareRequest struct {
	Share []ShareItem `json:"share"`
}

type ShareItem struct {
	ImageIDs []string     `json:"image_ids"`
	Config   *ShareConfig `json:"config,omitempty"`
}

type ShareConfig struct {
	// 1: on; 2: off (default).
	ShareStatus int `json:"share_status,omitempty"`
	// SharePhoneType is 1 (prefer subscription), 2 (subscription), or 3 (temporary).
	SharePhoneType int `json:"share_phone_type,omitempty"`
	// 8–20 characters, with uppercase/lowercase letters and digits.
	ShareCode string `json:"share_code,omitempty"`
	// 1: reset device; 2: factory reset; 3: edit; 4: cloud drive; 5: commands.
	ShareAuth []int `json:"share_auth,omitempty"`
}

type ShareResponse map[string]string

type InitProxyRequest struct {
	Images []InitProxyItem `json:"images"`
}

type InitProxyItem struct {
	ImageID string `json:"image_id"`
	// Same codes as PurchaseRequest.OS; required for an unconfigured Android version.
	OS   string `json:"os,omitempty"`
	Name string `json:"name,omitempty"`
	// IPScanChannel is required and is ip2location or ipapi.
	IPScanChannel string      `json:"ip_scan_channel"`
	Proxy         ProxyConfig `json:"proxy"`
	DPIName       string      `json:"dpi_name,omitempty"`
	// NetworkMode is 1 for Wi-Fi or 2 for mobile; Android 10, 11, and 12B only support Wi-Fi.
	NetworkMode int         `json:"network_mode,omitempty"`
	Brand       string      `json:"brand,omitempty"`
	Model       string      `json:"model,omitempty"`
	Location    *Location   `json:"location,omitempty"`
	SIM         *SIMConfig  `json:"sim,omitempty"`
	Locale      *InitLocale `json:"locale,omitempty"`
}

type ProxyConfig struct {
	ID       string `json:"id,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	// Protocol is socks5, http, or https.
	Protocol string `json:"protocol,omitempty"`
}

type Location struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

type SIMConfig struct {
	// Type is 1 (proxy IP), 2 (custom region), 3 (disabled), 4 (custom number), or 5 (cloud
	// number); omission defaults to 1.
	Type int `json:"type,omitempty"`
	// Country is required for Type 2 and uses ISO 3166-1 alpha-2.
	Country string `json:"country,omitempty"`
	// PhoneNumber is required for Type 4 or 5 and includes the leading +.
	PhoneNumber string `json:"phone_number,omitempty"`
}

type InitLocale struct {
	Country  string `json:"country"`
	Timezone string `json:"timezone"`
	Language string `json:"language,omitempty"`
}

type SwitchRequest struct {
	ImageIDs []string `json:"image_ids"`
	// Status is 0 for disabled or 1 for enabled.
	Status int `json:"status"`
}

type TimezoneListRequest struct {
	ISO string `json:"iso"`
}

type Option struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ResetDeviceRequest struct {
	ImageID string `json:"image_id"`
	ProxyID string `json:"proxy_id,omitempty"`
	// DNS is 1 for enabled or 2 for disabled; omission defaults to enabled.
	DNS    int           `json:"dns,omitempty"`
	Model  string        `json:"model,omitempty"`
	GPS    *GPSUpdate    `json:"gps,omitempty"`
	Locale *LocaleUpdate `json:"locale,omitempty"`
	SIM    *SIMConfig    `json:"sim,omitempty"`
	// 1: reinstall (default); 2: clear app data; 3: retain app data (CN; EN is ambiguous).
	DataType int    `json:"data_type,omitempty"`
	DPIName  string `json:"dpi_name,omitempty"`
	// NetworkMode is 1 for Wi-Fi or 2 for mobile; Android 10, 11, and 12B only support Wi-Fi.
	NetworkMode int `json:"network_mode,omitempty"`
	// KeepGP is 0 to clear Google Play or 1 to keep it; nil defaults to 0.
	KeepGP *int `json:"keep_gp,omitempty"`
}

type LiveRequest struct {
	ImageID string `json:"image_id"`
	// MP4 file ID; push to /sdcard/DCIM/Camera before enabling playback.
	ID string `json:"id,omitempty"`
	// 1: on; 2: off (default).
	Status int `json:"status,omitempty"`
	// 1: repeat (default); 2: no repeat.
	Loop int `json:"loop,omitempty"`
}

type ScanRequest struct {
	ImageID string `json:"image_id"`
	// ID is a cloud drive image file ID; push the file to /sdcard/DCIM/Camera first.
	ID string `json:"id"`
}

type ConvertVersionRequest struct {
	Images []ConvertVersionItem `json:"images"`
}

type ConvertVersionItem struct {
	ImageID string `json:"image_id"`
	// Same codes as PurchaseRequest.OS.
	OS string `json:"os"`
}

func (c *Client) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	var resp ListResponse
	if err := c.core.Do(ctx, pathList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PowerOn accepts at most 100 phones. Check Fail, then poll Status for readiness.
func (c *Client) PowerOn(ctx context.Context, imageIDs []string) (*common.OperationResult, error) {
	return c.operation(ctx, pathPowerOn, imageIDs)
}

// PowerOff accepts at most 20 phones.
func (c *Client) PowerOff(ctx context.Context, imageIDs []string) (*common.OperationResult, error) {
	return c.operation(ctx, pathPowerOff, imageIDs)
}

// Restart accepts at most 20 phones.
func (c *Client) Restart(ctx context.Context, imageIDs []string) (*common.OperationResult, error) {
	return c.operation(ctx, pathRestart, imageIDs)
}

func (c *Client) Status(ctx context.Context, imageIDs []string) (*StatusResponse, error) {
	var resp StatusResponse
	if err := c.core.Do(ctx, pathStatus, StatusRequest{ImageIDs: imageIDs}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Info(ctx context.Context, imageID string) (*Detail, error) {
	var resp Detail
	if err := c.core.Do(ctx, pathInfo, InfoRequest{ImageID: imageID}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Update(ctx context.Context, req UpdateRequest) (*common.OperationResult, error) {
	var resp common.OperationResult
	if err := c.core.Do(ctx, pathUpdate, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Models(ctx context.Context, req ModelListRequest) (ModelListResponse, error) {
	resp := ModelListResponse{}
	if err := c.core.Do(ctx, pathModelList, req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) BatchRoot(ctx context.Context, req BatchRootRequest) (*common.OperationResult, error) {
	var resp common.OperationResult
	if err := c.core.Do(ctx, pathBatchRoot, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Command runs a shell command without the adb shell prefix; maximum execution time is 10s.
func (c *Client) Command(ctx context.Context, imageID, command string) (*CommandResult, error) {
	var resp CommandResult
	if err := c.core.Do(ctx, pathCommand, CommandRequest{ImageID: imageID, Command: command}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CommandBatch runs a command on at most 20 phones; the same limits as Command apply.
func (c *Client) CommandBatch(ctx context.Context, imageIDs []string, command string) (map[string]CommandResult, error) {
	resp := map[string]CommandResult{}
	if err := c.core.Do(ctx, pathCommand, CommandRequest{ImageIDs: imageIDs, Command: command}, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) OpenADB(ctx context.Context, imageIDs []string) (*common.OperationResult, error) {
	return c.operation(ctx, pathOpenADB, imageIDs)
}

func (c *Client) CloseADB(ctx context.Context, imageIDs []string) (*common.OperationResult, error) {
	return c.operation(ctx, pathCloseADB, imageIDs)
}

func (c *Client) SetADBIPWhitelist(ctx context.Context, ips []string) (*common.MessageResponse, error) {
	return c.adbIPWhitelist(ctx, SetADBIPWhitelistRequest{IPs: ips})
}

func (c *Client) AppendADBIPWhitelist(ctx context.Context, ips []string) (*common.MessageResponse, error) {
	return c.adbIPWhitelist(ctx, SetADBIPWhitelistRequest{IPs: ips, Append: true})
}

func (c *Client) UpdateSharePassword(ctx context.Context, req UpdateSharePasswordRequest) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathUpdateSharePass, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) LinkUsers(ctx context.Context) (*LinkUsersResponse, error) {
	var resp LinkUsersResponse
	if err := c.core.Do(ctx, pathLinkUserList, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Tags(ctx context.Context, req TagListRequest) (*TagListResponse, error) {
	var resp TagListResponse
	if err := c.core.Do(ctx, pathTagList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Resources(ctx context.Context) (*ResourceListResponse, error) {
	var resp ResourceListResponse
	if err := c.core.Do(ctx, pathResourceList, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Resolutions(ctx context.Context) (*ResolutionListResponse, error) {
	var resp ResolutionListResponse
	if err := c.core.Do(ctx, pathResolutionList, nil, &resp); err != nil {
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

func (c *Client) Share(ctx context.Context, req ShareRequest) (ShareResponse, error) {
	resp := ShareResponse{}
	if err := c.core.Do(ctx, pathShare, req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) InitProxy(ctx context.Context, req InitProxyRequest) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathInitProxy, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SetProxyEnabled(ctx context.Context, imageIDs []string, enabled bool) (*common.OperationResult, error) {
	var resp common.OperationResult
	if err := c.core.Do(ctx, pathProxySwitch, SwitchRequest{ImageIDs: imageIDs, Status: common.BoolToInt(enabled)}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Timezones(ctx context.Context, iso string) ([]Option, error) {
	var resp []Option
	if err := c.core.Do(ctx, pathTimezoneList, TimezoneListRequest{ISO: iso}, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) Languages(ctx context.Context) ([]Option, error) {
	var resp []Option
	if err := c.core.Do(ctx, pathLanguageList, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ResetDevice requires a powered-on phone; DataType controls which data is removed.
func (c *Client) ResetDevice(ctx context.Context, req ResetDeviceRequest) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathNewPhone, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Live(ctx context.Context, req LiveRequest) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathLive, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ScanCode requires a file in /sdcard/DCIM/Camera and Android 12A, 15, or 15 Pro.
func (c *Client) ScanCode(ctx context.Context, imageID, fileID string) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathScan, ScanRequest{ImageID: imageID, ID: fileID}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SetAutoRenewal(ctx context.Context, imageIDs []string, enabled bool) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathAutoRenewal, SwitchRequest{ImageIDs: imageIDs, Status: common.BoolToInt(enabled)}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ConvertVersion(ctx context.Context, req ConvertVersionRequest) (*common.OperationResult, error) {
	var resp common.OperationResult
	if err := c.core.Do(ctx, pathConvertVersion, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) adbIPWhitelist(ctx context.Context, req SetADBIPWhitelistRequest) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathSetADBWhitelist, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) operation(ctx context.Context, path string, imageIDs []string) (*common.OperationResult, error) {
	var resp common.OperationResult
	if err := c.core.Do(ctx, path, StatusRequest{ImageIDs: imageIDs}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
