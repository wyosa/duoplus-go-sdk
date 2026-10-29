package duoplus

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wyosa/duoplus-go-sdk/automation"
	"github.com/wyosa/duoplus-go-sdk/clouddisk"
	"github.com/wyosa/duoplus-go-sdk/cloudnumber"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
	"github.com/wyosa/duoplus-go-sdk/common"
	"github.com/wyosa/duoplus-go-sdk/group"
	"github.com/wyosa/duoplus-go-sdk/proxy"
	"github.com/wyosa/duoplus-go-sdk/subscriptionstartup"
	"github.com/wyosa/duoplus-go-sdk/team"
)

type recorder struct {
	path string
	body []byte
	data string
}

func newRecordingClient(t *testing.T, data string) (*Client, *recorder) {
	t.Helper()
	rec := &recorder{data: data}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.path = r.URL.Path
		rec.body, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"code":200,"data":` + rec.data + `,"message":"Success"}`))
	}))
	t.Cleanup(srv.Close)

	client, err := NewClient("test-key", WithBaseURL(srv.URL), WithMinInterval(time.Nanosecond))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client, rec
}

func assertRequest(t *testing.T, rec *recorder, path, wantJSON string) {
	t.Helper()
	if rec.path != path {
		t.Errorf("path = %q, want %q", rec.path, path)
	}
	var got, want any
	if err := json.Unmarshal(rec.body, &got); err != nil {
		t.Fatalf("request body is not JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
		t.Fatalf("bad expectation: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("request body = %s, want %s", rec.body, wantJSON)
	}
}

func TestRequestBodies(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		call func(*Client) error
		path string
		want string
		data string
	}{
		{
			name: "cloud phone update clears optional fields",
			call: func(c *Client) error {
				_, err := c.CloudPhones.Update(ctx, cloudphone.UpdateRequest{Images: []cloudphone.UpdateItem{{
					ImageID: "p1", Remark: common.Ptr(""),
					Proxy:     &cloudphone.ProxyUpdate{ID: common.Ptr("")},
					Locale:    &cloudphone.LocaleUpdate{Language: common.Ptr("")},
					SIM:       &cloudphone.SIMUpdate{NumberID: common.Ptr("")},
					Bluetooth: &cloudphone.BluetoothUpdate{Name: common.Ptr("")},
					WiFi:      &cloudphone.WiFiUpdate{Name: common.Ptr("")},
					Device:    &cloudphone.DeviceUpdate{Name: common.Ptr("")},
					Station:   &cloudphone.StationUpdate{LAC: common.Ptr(0), CID: common.Ptr(0)},
				}}})
				return err
			},
			path: "/api/v1/cloudPhone/update",
			want: `{"images":[{"image_id":"p1","remark":"","proxy":{"id":""},"locale":{"language":""},"sim":{"number_id":""},"bluetooth":{"name":""},"wifi":{"name":""},"device":{"name":""},"station":{"lac":0,"cid":0}}]}`,
		},
		{
			name: "cloud phone update omits unchanged fields",
			call: func(c *Client) error {
				_, err := c.CloudPhones.Update(ctx, cloudphone.UpdateRequest{Images: []cloudphone.UpdateItem{{ImageID: "p1", Name: common.Ptr("Renamed")}}})
				return err
			},
			path: "/api/v1/cloudPhone/update",
			want: `{"images":[{"image_id":"p1","name":"Renamed"}]}`,
		},
		{
			name: "proxy update clears credentials and refresh URL",
			call: func(c *Client) error {
				_, err := c.Proxies.Update(ctx, proxy.UpdateRequest{ID: "x", User: common.Ptr(""), Password: common.Ptr(""), RefreshURL: common.Ptr("")})
				return err
			},
			path: "/api/v1/proxy/update",
			want: `{"id":"x","user":"","password":"","refresh_url":""}`,
		},
		{
			name: "plan update clears remark without changing name",
			call: func(c *Client) error {
				_, err := c.Automation.SavePlan(ctx, automation.SavePlanRequest{ID: "plan1", Remark: common.Ptr(""), Images: []automation.TaskImage{{ImageID: "p1"}}})
				return err
			},
			path: "/api/v1/automation/savePlan",
			want: `{"id":"plan1","remark":"","images":[{"image_id":"p1"}]}`,
		},
		{
			name: "cloud phone purchase without auto-renewal",
			call: func(c *Client) error {
				_, err := c.CloudPhones.Purchase(ctx, cloudphone.PurchaseRequest{Quantity: 1, RenewalStatus: common.Ptr(0)})
				return err
			},
			path: "/api/v1/cloudPhone/purchase",
			want: `{"quantity":1,"renewal_status":0}`,
		},
		{
			name: "adb whitelist overwrite",
			call: func(c *Client) error {
				_, err := c.CloudPhones.SetADBIPWhitelist(ctx, []string{"1.1.1.1"})
				return err
			},
			path: "/api/v1/cloudPhone/setAdbIpWhitelist",
			want: `{"ips":["1.1.1.1"]}`,
		},
		{
			name: "adb whitelist append",
			call: func(c *Client) error {
				_, err := c.CloudPhones.AppendADBIPWhitelist(ctx, []string{"1.1.1.1"})
				return err
			},
			path: "/api/v1/cloudPhone/setAdbIpWhitelist",
			want: `{"ips":["1.1.1.1"],"append":true}`,
		},
		{
			name: "cloud phone update with zero coordinates, station and cloud number",
			call: func(c *Client) error {
				_, err := c.CloudPhones.Update(ctx, cloudphone.UpdateRequest{Images: []cloudphone.UpdateItem{{
					ImageID: "p1",
					GPS:     &cloudphone.GPSUpdate{Type: 2, Longitude: common.Ptr(0.0), Latitude: common.Ptr(51.5)},
					SIM:     &cloudphone.SIMUpdate{NumberID: common.Ptr("n1")},
					Station: &cloudphone.StationUpdate{LAC: common.Ptr(55), CID: common.Ptr(18)},
				}}})
				return err
			},
			path: "/api/v1/cloudPhone/update",
			want: `{"images":[{"image_id":"p1","gps":{"type":2,"longitude":0,"latitude":51.5},"sim":{"number_id":"n1"},"station":{"lac":55,"cid":18}}]}`,
		},
		{
			name: "subscription startup purchase without auto-renewal",
			call: func(c *Client) error {
				_, err := c.SubscriptionStartups.Purchase(ctx, subscriptionstartup.PurchaseRequest{Duration: "30", Quantity: 1, RenewalStatus: common.Ptr(0)})
				return err
			},
			path: "/api/v1/subscriptionStartup/purchase",
			want: `{"duration":"30","quantity":1,"renewal_status":0}`,
		},
		{
			name: "subscription startup list filters",
			call: func(c *Client) error {
				_, err := c.SubscriptionStartups.List(ctx, subscriptionstartup.ListRequest{RenewalStatus: common.Ptr(0), FreeStatus: common.Ptr(1)})
				return err
			},
			path: "/api/v1/subscriptionStartup/list",
			want: `{"renewal_status":0,"free_status":1}`,
		},
		{
			name: "group create with zero sort",
			call: func(c *Client) error {
				_, err := c.Groups.Create(ctx, group.CreateRequest{List: []group.CreateItem{{Name: "g1", Sort: common.Ptr(0)}}})
				return err
			},
			path: "/api/v1/cloudPhone/createGroup",
			want: `{"list":[{"name":"g1","sort":0}]}`,
		},
		{
			name: "group update clears remark",
			call: func(c *Client) error {
				_, err := c.Groups.Update(ctx, group.UpdateRequest{List: []group.UpdateItem{{ID: "1", Name: "g1", Remark: common.Ptr("")}}})
				return err
			},
			path: "/api/v1/cloudPhone/updateGroup",
			want: `{"list":[{"id":"1","name":"g1","remark":""}]}`,
		},
		{
			name: "proxy update protocol",
			call: func(c *Client) error {
				_, err := c.Proxies.Update(ctx, proxy.UpdateRequest{ID: "x", Protocol: common.Ptr("http")})
				return err
			},
			path: "/api/v1/proxy/update",
			want: `{"id":"x","protocol":"http"}`,
		},
		{
			name: "cloud phone info follows both request examples",
			call: func(c *Client) error {
				_, err := c.CloudPhones.Info(ctx, "p1")
				return err
			},
			path: "/api/v1/cloudPhone/info",
			want: `{"image_id":"p1"}`,
		},
		{
			name: "batch root follows both request examples",
			call: func(c *Client) error {
				_, err := c.CloudPhones.BatchRoot(ctx, cloudphone.BatchRootRequest{ImageIDs: []string{"p1"}, Status: 3, Packages: []string{"com.a"}})
				return err
			},
			path: "/api/v1/cloudPhone/batchRoot",
			want: `{"image_ids":["p1"],"status":3,"packages":["com.a"]}`,
		},
		{
			name: "write sms sends image_id as string",
			call: func(c *Client) error {
				_, err := c.CloudNumbers.WriteSMS(ctx, cloudnumber.WriteSMSRequest{ImageID: "p1", SMS: []cloudnumber.SMSItem{{Phone: "+1", Message: "hi"}}})
				return err
			},
			path: "/api/v1/cloudNumber/imageWriteSms",
			want: `{"image_id":"p1","sms":[{"phone":"+1","message":"hi"}]}`,
		},
		{
			name: "init proxy",
			call: func(c *Client) error {
				_, err := c.CloudPhones.InitProxy(ctx, cloudphone.InitProxyRequest{Images: []cloudphone.InitProxyItem{{
					ImageID:       "p1",
					IPScanChannel: "ip2location",
					Proxy:         cloudphone.ProxyConfig{Host: "1.2.3.4", Port: 1080},
					NetworkMode:   2,
					Location:      &cloudphone.Location{Longitude: 0, Latitude: 51.5},
					SIM:           &cloudphone.SIMConfig{Type: 2, Country: "GB"},
					Locale:        &cloudphone.InitLocale{Country: "GB", Timezone: "Europe/London", Language: "en-GB"},
				}}})
				return err
			},
			path: "/api/v1/cloudPhone/initProxy",
			want: `{"images":[{"image_id":"p1","ip_scan_channel":"ip2location","proxy":{"host":"1.2.3.4","port":1080},"network_mode":2,
				"location":{"longitude":0,"latitude":51.5},"sim":{"type":2,"country":"GB"},"locale":{"country":"GB","timezone":"Europe/London","language":"en-GB"}}]}`,
		},
		{
			name: "disable cloud phone proxy",
			call: func(c *Client) error {
				_, err := c.CloudPhones.SetProxyEnabled(ctx, []string{"p1"}, false)
				return err
			},
			path: "/api/v1/cloudPhone/proxySwitch",
			want: `{"image_ids":["p1"],"status":0}`,
		},
		{
			name: "timezones",
			call: func(c *Client) error {
				_, err := c.CloudPhones.Timezones(ctx, "GB")
				return err
			},
			path: "/api/v1/mobile/timezoneList",
			want: `{"iso":"GB"}`,
			data: `[]`,
		},
		{
			name: "reset device keeping nothing",
			call: func(c *Client) error {
				_, err := c.CloudPhones.ResetDevice(ctx, cloudphone.ResetDeviceRequest{ImageID: "p1", KeepGP: common.Ptr(0), GPS: &cloudphone.GPSUpdate{Type: 1}})
				return err
			},
			path: "/api/v1/cloudPhone/newPhone",
			want: `{"image_id":"p1","gps":{"type":1},"keep_gp":0}`,
		},
		{
			name: "live",
			call: func(c *Client) error {
				_, err := c.CloudPhones.Live(ctx, cloudphone.LiveRequest{ImageID: "p1", ID: "f1", Status: 1, Loop: 2})
				return err
			},
			path: "/api/v1/cloudPhone/live",
			want: `{"image_id":"p1","id":"f1","status":1,"loop":2}`,
		},
		{
			name: "scan code",
			call: func(c *Client) error {
				_, err := c.CloudPhones.ScanCode(ctx, "p1", "f1")
				return err
			},
			path: "/api/v1/cloudPhone/scan",
			want: `{"image_id":"p1","id":"f1"}`,
		},
		{
			name: "disable cloud phone auto-renewal",
			call: func(c *Client) error {
				_, err := c.CloudPhones.SetAutoRenewal(ctx, []string{"p1"}, false)
				return err
			},
			path: "/api/v1/cloudPhone/autoRenewal",
			want: `{"image_ids":["p1"],"status":0}`,
		},
		{
			name: "convert version",
			call: func(c *Client) error {
				_, err := c.CloudPhones.ConvertVersion(ctx, cloudphone.ConvertVersionRequest{Images: []cloudphone.ConvertVersionItem{{ImageID: "p1", OS: "15pro"}}})
				return err
			},
			path: "/api/v1/cloudPhone/convertVersion",
			want: `{"images":[{"image_id":"p1","os":"15pro"}]}`,
		},
		{
			name: "disable subscription startup auto-renewal",
			call: func(c *Client) error {
				_, err := c.SubscriptionStartups.SetAutoRenewal(ctx, []string{"s1"}, false)
				return err
			},
			path: "/api/v1/subscriptionStartup/autoRenewal",
			want: `{"phone_ids":["s1"],"status":0}`,
		},
		{
			name: "check saved proxy",
			call: func(c *Client) error {
				_, err := c.Proxies.Check(ctx, proxy.CheckRequest{ProxyID: "x"})
				return err
			},
			path: "/api/v1/proxy/check",
			want: `{"proxy_id":"x"}`,
		},
		{
			name: "enable proxy auto-renewal",
			call: func(c *Client) error {
				_, err := c.Proxies.SetAutoRenewal(ctx, []string{"x"}, true)
				return err
			},
			path: "/api/v1/proxy/autoRenewal",
			want: `{"proxy_ids":["x"],"type":1}`,
		},
		{
			name: "cloud number package durations for VOIP",
			call: func(c *Client) error {
				_, err := c.CloudNumbers.PackageDurations(ctx, cloudnumber.PackageDurationsRequest{Region: "US", Type: common.Ptr(0)})
				return err
			},
			path: "/api/v1/cloudNumber/package",
			want: `{"region":"US","type":0}`,
		},
		{
			name: "cloud number purchase",
			call: func(c *Client) error {
				_, err := c.CloudNumbers.Purchase(ctx, cloudnumber.PurchaseRequest{Region: "US", Duration: 30, Type: common.Ptr(0), RenewalStatus: common.Ptr(0)})
				return err
			},
			path: "/api/v1/cloudNumber/purchase",
			want: `{"region":"US","duration":30,"type":0,"renewal_status":0}`,
		},
		{
			name: "cloud number renewal packages",
			call: func(c *Client) error {
				_, err := c.CloudNumbers.RenewalPackages(ctx, []string{"n1"})
				return err
			},
			path: "/api/v1/cloudNumber/renewalPackage",
			want: `{"number_ids":["n1"]}`,
		},
		{
			name: "cloud number renew",
			call: func(c *Client) error {
				_, err := c.CloudNumbers.Renew(ctx, cloudnumber.RenewRequest{List: []cloudnumber.RenewItem{{NumberIDs: []string{"n1"}, Duration: 30}}})
				return err
			},
			path: "/api/v1/cloudNumber/renewal",
			want: `{"list":[{"number_ids":["n1"],"duration":30}]}`,
		},
		{
			name: "disable cloud number auto-renewal",
			call: func(c *Client) error {
				_, err := c.CloudNumbers.SetAutoRenewal(ctx, []string{"n1"}, false)
				return err
			},
			path: "/api/v1/cloudNumber/autoRenewal",
			want: `{"number_ids":["n1"],"type":0}`,
		},
		{
			name: "delete cloud drive files",
			call: func(c *Client) error {
				_, err := c.CloudDisk.Delete(ctx, []string{"f1"})
				return err
			},
			path: "/api/v1/cloudDisk/delFiles",
			want: `{"ids":["f1"]}`,
		},
		{
			name: "official template info",
			call: func(c *Client) error {
				_, err := c.Automation.OfficialTemplateInfo(ctx, "mgRg1")
				return err
			},
			path: "/api/v1/automation/officialTemplateInfo",
			want: `{"id":"mgRg1"}`,
		},
		{
			name: "user template info",
			call: func(c *Client) error {
				_, err := c.Automation.UserTemplateInfo(ctx, "mgRg1")
				return err
			},
			path: "/api/v1/automation/userTemplateInfo",
			want: `{"id":"mgRg1"}`,
		},
		{
			name: "team orders",
			call: func(c *Client) error {
				_, err := c.Team.Orders(ctx, team.OrderListRequest{Status: []string{"1"}, PaginationRequest: common.PaginationRequest{Page: 1, PageSize: 500}})
				return err
			},
			path: "/api/v1/team/order",
			want: `{"status":["1"],"page":1,"pagesize":500}`,
		},
		{
			name: "team balance",
			call: func(c *Client) error {
				_, err := c.Team.Balance(ctx)
				return err
			},
			path: "/api/v1/team/balance",
			want: `{}`,
		},
		{
			name: "cloud number list by ids",
			call: func(c *Client) error {
				_, err := c.CloudNumbers.List(ctx, cloudnumber.ListRequest{IDs: []string{"a", "b"}})
				return err
			},
			path: "/api/v1/cloudNumber/numberList",
			want: `{"id":["a","b"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.data
			if data == "" {
				data = `{}`
			}
			client, rec := newRecordingClient(t, data)
			if err := tt.call(client); err != nil {
				t.Fatalf("call error = %v", err)
			}
			assertRequest(t, rec, tt.path, tt.want)
		})
	}
}

func TestCloudPhoneListDecodesNewFields(t *testing.T) {
	client, _ := newRecordingClient(t, `{"list":[{"id":"p1","adb_status":1,"http_status":0,"region":"US",
		"group":[{"id":"g1","name":"Group"}],"android_version_configured":true,"new_mock_times":3}],"page":1,"pagesize":10,"total":1,"total_page":1}`)

	resp, err := client.CloudPhones.List(context.Background(), cloudphone.ListRequest{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	p := resp.List[0]
	if p.ADBStatus != 1 || p.HTTPStatus != "0" || p.Region != "US" || !p.AndroidVersionConfigured || p.NewMockTimes != 3 {
		t.Fatalf("unexpected phone %+v", p)
	}
	if len(p.Group) != 1 || p.Group[0].Name != "Group" {
		t.Fatalf("unexpected group %+v", p.Group)
	}
}

func TestOperationResultAcceptsArrayFailReason(t *testing.T) {
	client, _ := newRecordingClient(t, `{"success":["p1"],"fail":[],"fail_reason":[]}`)

	resp, err := client.CloudPhones.OpenADB(context.Background(), []string{"p1"})
	if err != nil {
		t.Fatalf("OpenADB() error = %v", err)
	}
	if len(resp.Success) != 1 || len(resp.FailReason) != 0 {
		t.Fatalf("unexpected result %+v", resp)
	}
}

func TestOrderResponseDecodesProductIDs(t *testing.T) {
	client, _ := newRecordingClient(t, `{"order_id":"o1","product_ids":["a","b"]}`)

	resp, err := client.SubscriptionStartups.Purchase(context.Background(), subscriptionstartup.PurchaseRequest{Quantity: 2})
	if err != nil {
		t.Fatalf("Purchase() error = %v", err)
	}
	if resp.OrderID != "o1" || !reflect.DeepEqual(resp.ProductIDs, []string{"a", "b"}) {
		t.Fatalf("unexpected order %+v", resp)
	}
}

func TestOfficialTemplatesDecodeExtraFields(t *testing.T) {
	client, _ := newRecordingClient(t, `{"list":[{"id":"t1","name":"Login","desc":"d","version":["45.3.3"],
		"lang":["English"],"is_ai_rpa":1,"is_pinned":true,"pinned_at":"2026-09-24 15:46:14.433"}]}`)

	resp, err := client.Automation.ListOfficialTemplates(context.Background(), automation.TemplateListRequest{})
	if err != nil {
		t.Fatalf("ListOfficialTemplates() error = %v", err)
	}
	tpl := resp.List[0]
	if tpl.ID != "t1" || tpl.Version[0] != "45.3.3" || tpl.IsAIRPA != 1 || !tpl.IsPinned {
		t.Fatalf("unexpected template %+v", tpl)
	}
}

func TestListResponsesDecodeNewFields(t *testing.T) {
	client, _ := newRecordingClient(t, `{"list":[{"id":"x","group_ids":["g1"],"group_name":["G"],"assigned":true,"free_status":1}]}`)
	ctx := context.Background()

	proxies, err := client.Proxies.List(ctx, proxy.ListRequest{})
	if err != nil {
		t.Fatalf("Proxies.List() error = %v", err)
	}
	if !reflect.DeepEqual(proxies.List[0].GroupIDs, []string{"g1"}) || !reflect.DeepEqual(proxies.List[0].GroupName, []string{"G"}) {
		t.Fatalf("unexpected proxy %+v", proxies.List[0])
	}

	numbers, err := client.CloudNumbers.List(ctx, cloudnumber.ListRequest{})
	if err != nil {
		t.Fatalf("CloudNumbers.List() error = %v", err)
	}
	if !numbers.List[0].Assigned {
		t.Fatalf("unexpected number %+v", numbers.List[0])
	}

	subs, err := client.SubscriptionStartups.List(ctx, subscriptionstartup.ListRequest{})
	if err != nil {
		t.Fatalf("SubscriptionStartups.List() error = %v", err)
	}
	if subs.List[0].FreeStatus != 1 {
		t.Fatalf("unexpected subscription startup %+v", subs.List[0])
	}
}

func TestArrayDataResponses(t *testing.T) {
	ctx := context.Background()

	client, _ := newRecordingClient(t, `[{"id":"Europe/London","name":"Europe/London(GMT+01:00)"}]`)
	zones, err := client.CloudPhones.Timezones(ctx, "GB")
	if err != nil {
		t.Fatalf("Timezones() error = %v", err)
	}
	if len(zones) != 1 || zones[0].ID != "Europe/London" {
		t.Fatalf("unexpected timezones %+v", zones)
	}

	client, _ = newRecordingClient(t, `[{"type":0,"name":"VOIP","country":[{"region":"US","name":"US",
		"durations":[{"name":"30 days","days":30,"random_gif_days":"25-50","price":4.9,"total_price":4.9}]}]}]`)
	pkgs, err := client.CloudNumbers.PurchasePackages(ctx)
	if err != nil {
		t.Fatalf("PurchasePackages() error = %v", err)
	}
	if d := pkgs[0].Country[0].Durations[0]; d.Days != 30 || d.RandomGifDays != "25-50" || d.TotalPrice != 4.9 {
		t.Fatalf("unexpected packages %+v", pkgs)
	}

	// The API sends [] for some empty objects.
	client, _ = newRecordingClient(t, `[]`)
	models, err := client.CloudPhones.Models(ctx, cloudphone.ModelListRequest{OS: 1})
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("unexpected models %+v", models)
	}

	client, _ = newRecordingClient(t, `{"id":"p1","proxy":[],"device":[],"group":[]}`)
	detail, err := client.CloudPhones.Info(ctx, "p1")
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if detail.ID != "p1" || detail.Proxy.ID != "" {
		t.Fatalf("unexpected detail %+v", detail)
	}
}

func TestWrappedResponses(t *testing.T) {
	ctx := context.Background()

	client, _ := newRecordingClient(t, `{"balance":{"balance":"$73.00","recharge_balance":"$70.00","gift_balance":"$3.00"}}`)
	balance, err := client.Team.Balance(ctx)
	if err != nil {
		t.Fatalf("Balance() error = %v", err)
	}
	if balance.Balance != "$73.00" || balance.GiftBalance != "$3.00" {
		t.Fatalf("unexpected balance %+v", balance)
	}

	client, _ = newRecordingClient(t, `{"info":{"id":"mgRg1","app_id":"qunBQ","config":[{"key":"username","value":"u","type":"string","required":true,"desc":"d"}]}}`)
	tpl, err := client.Automation.OfficialTemplateInfo(ctx, "mgRg1")
	if err != nil {
		t.Fatalf("OfficialTemplateInfo() error = %v", err)
	}
	if tpl.AppID != "qunBQ" || len(tpl.Config) != 1 || tpl.Config[0].Desc != "d" {
		t.Fatalf("unexpected template %+v", tpl)
	}

	client, _ = newRecordingClient(t, `{"success":false,"tips":"Connection test failed!"}`)
	check, err := client.Proxies.Check(ctx, proxy.CheckRequest{ProxyID: "x"})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if check.Success || check.Tips == "" {
		t.Fatalf("unexpected check %+v", check)
	}
}

func TestCloudDiskUpload(t *testing.T) {
	var (
		gotMethod, gotCallback, gotKey string
		gotBody                        []byte
		gotLength                      int64
	)
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/api/v1/cloudDisk/signedUrl", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"name":"test.apk","is_app":1,"pkg":"com.test"}` {
			t.Errorf("signedUrl body = %s", body)
		}
		w.Write([]byte(`{"code":200,"data":{"method":"PUT","signedUrl":"` + srv.URL + `/oss/test.apk?Signature=abc",
			"headers":{"x-oss-callback":"cb","x-oss-callback-var":"vars"},"name":"test.apk","original_file_name":"test.apk"},"message":"Success"}`))
	})
	mux.HandleFunc("/oss/test.apk", func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCallback = r.Header.Get("x-oss-callback")
		gotKey = r.Header.Get("DuoPlus-API-Key")
		gotLength = r.ContentLength
		gotBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"code":200,"message":"Success"}`))
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient("test-key", WithBaseURL(srv.URL), WithMinInterval(time.Nanosecond))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	content := "apk-bytes"
	resp, err := client.CloudDisk.Upload(context.Background(), clouddisk.SignedURLRequest{Name: "test.apk", IsApp: 1, Pkg: "com.test"},
		strings.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if resp.Name != "test.apk" {
		t.Fatalf("unexpected response %+v", resp)
	}
	if gotMethod != http.MethodPut || gotCallback != "cb" || string(gotBody) != content || gotLength != int64(len(content)) {
		t.Fatalf("unexpected upload: method=%s callback=%q body=%q length=%d", gotMethod, gotCallback, gotBody, gotLength)
	}
	if gotKey != "" {
		t.Fatal("api key was sent to the storage url")
	}
}

func TestCloudDiskUploadErrorHidesSignedURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":200,"data":{"method":"PUT","signedUrl":"https://127.0.0.1:1/x?Signature=secret-sig","headers":{}},"message":"ok"}`))
	}))
	defer srv.Close()

	client, err := NewClient("test-key", WithBaseURL(srv.URL), WithMinInterval(time.Nanosecond))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.CloudDisk.Upload(context.Background(), clouddisk.SignedURLRequest{Name: "a.txt"}, strings.NewReader("x"), 1)
	if err == nil {
		t.Fatal("expected upload error")
	}
	if strings.Contains(err.Error(), "secret-sig") {
		t.Fatalf("error leaks signed url: %v", err)
	}
}
