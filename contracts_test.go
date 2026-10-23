package duoplus_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/app"
	"github.com/wyosa/duoplus-go-sdk/automation"
	"github.com/wyosa/duoplus-go-sdk/clouddisk"
	"github.com/wyosa/duoplus-go-sdk/cloudnumber"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
	"github.com/wyosa/duoplus-go-sdk/group"
	"github.com/wyosa/duoplus-go-sdk/proxy"
	"github.com/wyosa/duoplus-go-sdk/subscriptionstartup"
	"github.com/wyosa/duoplus-go-sdk/team"
)

type contractFixture struct {
	Name      string          `json:"name"`
	Operation string          `json:"operation"`
	Path      string          `json:"path"`
	Source    string          `json:"source"`
	CheckedAt string          `json:"checked_at"`
	Notes     string          `json:"notes"`
	Request   json.RawMessage `json:"request"`
	Response  json.RawMessage `json:"response"`
	Result    json.RawMessage `json:"result"`
}

func contractFixtures(t *testing.T) []contractFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []contractFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	return fixtures
}

// Requests match exactly; responses must preserve every fixture field.
func TestPublishedContracts(t *testing.T) {
	for _, fixture := range contractFixtures(t) {
		t.Run(fixture.Name, func(t *testing.T) {
			client, calls := contractClient(t, fixture, http.StatusOK, fixture.Response)
			result, err := invokeContract(t, client, fixture)
			if err != nil {
				t.Fatalf("%s: %v (source: %s)", fixture.Operation, err, fixture.Source)
			}
			if calls.Load() != 1 {
				t.Fatalf("sent %d requests, want 1", calls.Load())
			}
			expected := fixture.Result
			if len(expected) == 0 {
				var envelope struct {
					Data json.RawMessage `json:"data"`
				}
				if err := json.Unmarshal(fixture.Response, &envelope); err != nil {
					t.Fatal(err)
				}
				expected = envelope.Data
			}
			got, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			assertContractProjection(t, decodeContractJSON(t, got), decodeContractJSON(t, expected), "data")
		})
	}
}

func TestServiceMethodsPreserveAPIErrors(t *testing.T) {
	seen := make(map[string]bool)
	for _, fixture := range contractFixtures(t) {
		if seen[fixture.Operation] {
			continue
		}
		seen[fixture.Operation] = true
		t.Run(fixture.Operation, func(t *testing.T) {
			client, _ := contractClient(t, fixture, http.StatusServiceUnavailable,
				[]byte(`{"code":503,"data":null,"message":"fixture failure"}`))
			result, err := invokeContract(t, client, fixture)
			var apiErr *duoplus.APIError
			if !errors.As(err, &apiErr) || apiErr.Code != 503 || apiErr.HTTPStatus != 503 || apiErr.Message != "fixture failure" {
				t.Fatalf("lost server error: %v", err)
			}
			if result != nil && !reflect.ValueOf(result).IsNil() {
				t.Fatalf("returned a result with an error: %#v", result)
			}
		})
	}
}

func contractClient(t *testing.T, fixture contractFixture, status int, response []byte) (*duoplus.Client, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	wantRequest := decodeContractJSON(t, fixture.Request)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != fixture.Path {
			t.Errorf("request = %s %s, want POST %s", r.Method, r.URL.Path, fixture.Path)
		}
		if r.Header.Get("DuoPlus-API-Key") != "fixture-key" || r.Header.Get("Lang") != "en" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing or incorrect API headers")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		var gotRequest any
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.UseNumber()
		if err := decoder.Decode(&gotRequest); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if !reflect.DeepEqual(gotRequest, wantRequest) {
			t.Errorf("request = %s, want %s", body, fixture.Request)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(response)
	}))
	t.Cleanup(srv.Close)
	client, err := duoplus.NewClient("fixture-key", duoplus.WithBaseURL(srv.URL), duoplus.WithMinInterval(time.Nanosecond))
	if err != nil {
		t.Fatal(err)
	}
	return client, &calls
}

func decodeContractRequest[T any](t *testing.T, data []byte) T {
	t.Helper()
	var request T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(&request); err != nil {
		t.Fatalf("fixture cannot be represented by the SDK request: %v", err)
	}
	return request
}

func decodeContractJSON(t *testing.T, data []byte) any {
	t.Helper()
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func assertContractProjection(t *testing.T, got, want any, path string) {
	t.Helper()
	switch want := want.(type) {
	case map[string]any:
		actual, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("%s: got %T, want object", path, got)
		}
		for key, value := range want {
			next, exists := actual[key]
			if !exists {
				// Shared batch results omit empty success/fail/fail_reason collections.
				if (key == "success" || key == "fail" || key == "fail_reason") && emptyContractCollection(value) {
					continue
				}
				t.Errorf("%s.%s: documented field was lost", path, key)
				continue
			}
			assertContractProjection(t, next, value, path+"."+key)
		}
	case []any:
		actual, ok := got.([]any)
		if got == nil && len(want) == 0 {
			return
		}
		if !ok || len(actual) != len(want) {
			t.Fatalf("%s: got %v, want array of length %d", path, got, len(want))
		}
		for i, value := range want {
			assertContractProjection(t, actual[i], value, fmt.Sprintf("%s[%d]", path, i))
		}
	default:
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %#v, want %#v", path, got, want)
		}
	}
}

func emptyContractCollection(value any) bool {
	switch value := value.(type) {
	case []any:
		return len(value) == 0
	case map[string]any:
		return len(value) == 0
	default:
		return false
	}
}

// Fail when a new endpoint or service method lacks a sourced contract fixture.
func TestContractCatalogCoversSDKEndpoints(t *testing.T) {
	covered := make(map[string]bool)
	operations := make(map[string]bool)
	names := make(map[string]bool)
	for _, fixture := range contractFixtures(t) {
		if names[fixture.Name] {
			t.Errorf("duplicate fixture %s", fixture.Name)
		}
		names[fixture.Name] = true
		covered[fixture.Path] = true
		operations[fixture.Operation] = true
		if !strings.HasPrefix(fixture.Source, "https://help.duoplus.net/docs/") && !strings.HasPrefix(fixture.Source, "https://help.duoplus.cn/docs/") {
			t.Errorf("%s: missing official source", fixture.Name)
		}
		if _, err := time.Parse(time.DateOnly, fixture.CheckedAt); err != nil {
			t.Errorf("%s: invalid review date", fixture.Name)
		}
	}
	clientType := reflect.TypeOf(duoplus.Client{})
	methods := make(map[string]bool)
	for i := 0; i < clientType.NumField(); i++ {
		service := clientType.Field(i)
		if !service.IsExported() {
			continue
		}
		for j := 0; j < service.Type.NumMethod(); j++ {
			operation := service.Name + "." + service.Type.Method(j).Name
			// These compose API and storage requests and have dedicated upload tests.
			if operation == "CloudDisk.Upload" || operation == "CloudDisk.UploadFile" {
				continue
			}
			methods[operation] = true
			if !operations[operation] {
				t.Errorf("service method has no contract fixture: %s", operation)
			}
		}
	}
	for operation := range operations {
		if !methods[operation] {
			t.Errorf("fixture has no public service method: %s", operation)
		}
	}
	files, err := filepath.Glob("*/client.go")
	if err != nil {
		t.Fatal(err)
	}
	implemented := make(map[string]bool)
	for _, file := range files {
		tree, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(tree, func(node ast.Node) bool {
			declaration, ok := node.(*ast.GenDecl)
			if !ok || declaration.Tok != token.CONST {
				return true
			}
			for _, spec := range declaration.Specs {
				for _, value := range spec.(*ast.ValueSpec).Values {
					literal, ok := value.(*ast.BasicLit)
					if !ok || literal.Kind != token.STRING {
						continue
					}
					path, err := strconv.Unquote(literal.Value)
					if err == nil && strings.HasPrefix(path, "/api/v1/") {
						implemented[path] = true
					}
				}
			}
			return false
		})
	}
	if len(implemented) == 0 {
		t.Fatal("no endpoint declarations found")
	}
	for path := range implemented {
		if !covered[path] {
			t.Errorf("endpoint has no contract fixture: %s", path)
		}
	}
	for path := range covered {
		if !implemented[path] {
			t.Errorf("fixture has no implemented endpoint: %s", path)
		}
	}
}

func invokeContract(t *testing.T, client *duoplus.Client, fixture contractFixture) (any, error) {
	t.Helper()
	ctx := context.Background()
	switch fixture.Operation {
	case "CloudPhones.List":
		return client.CloudPhones.List(ctx, decodeContractRequest[cloudphone.ListRequest](t, fixture.Request))
	case "CloudPhones.PowerOn":
		req := decodeContractRequest[cloudphone.StatusRequest](t, fixture.Request)
		return client.CloudPhones.PowerOn(ctx, req.ImageIDs)
	case "CloudPhones.PowerOff":
		req := decodeContractRequest[cloudphone.StatusRequest](t, fixture.Request)
		return client.CloudPhones.PowerOff(ctx, req.ImageIDs)
	case "CloudPhones.Restart":
		req := decodeContractRequest[cloudphone.StatusRequest](t, fixture.Request)
		return client.CloudPhones.Restart(ctx, req.ImageIDs)
	case "CloudPhones.Status":
		req := decodeContractRequest[cloudphone.StatusRequest](t, fixture.Request)
		return client.CloudPhones.Status(ctx, req.ImageIDs)
	case "CloudPhones.Info":
		req := decodeContractRequest[cloudphone.InfoRequest](t, fixture.Request)
		return client.CloudPhones.Info(ctx, req.ImageID)
	case "CloudPhones.Update":
		return client.CloudPhones.Update(ctx, decodeContractRequest[cloudphone.UpdateRequest](t, fixture.Request))
	case "CloudPhones.Models":
		return client.CloudPhones.Models(ctx, decodeContractRequest[cloudphone.ModelListRequest](t, fixture.Request))
	case "CloudPhones.BatchRoot":
		return client.CloudPhones.BatchRoot(ctx, decodeContractRequest[cloudphone.BatchRootRequest](t, fixture.Request))
	case "CloudPhones.Command":
		req := decodeContractRequest[cloudphone.CommandRequest](t, fixture.Request)
		return client.CloudPhones.Command(ctx, req.ImageID, req.Command)
	case "CloudPhones.CommandBatch":
		req := decodeContractRequest[cloudphone.CommandRequest](t, fixture.Request)
		return client.CloudPhones.CommandBatch(ctx, req.ImageIDs, req.Command)
	case "CloudPhones.OpenADB":
		req := decodeContractRequest[cloudphone.StatusRequest](t, fixture.Request)
		return client.CloudPhones.OpenADB(ctx, req.ImageIDs)
	case "CloudPhones.CloseADB":
		req := decodeContractRequest[cloudphone.StatusRequest](t, fixture.Request)
		return client.CloudPhones.CloseADB(ctx, req.ImageIDs)
	case "CloudPhones.UpdateSharePassword":
		return client.CloudPhones.UpdateSharePassword(ctx, decodeContractRequest[cloudphone.UpdateSharePasswordRequest](t, fixture.Request))
	case "CloudPhones.LinkUsers":
		return client.CloudPhones.LinkUsers(ctx)
	case "CloudPhones.Tags":
		return client.CloudPhones.Tags(ctx, decodeContractRequest[cloudphone.TagListRequest](t, fixture.Request))
	case "CloudPhones.Resources":
		return client.CloudPhones.Resources(ctx)
	case "CloudPhones.Resolutions":
		return client.CloudPhones.Resolutions(ctx)
	case "CloudPhones.Purchase":
		return client.CloudPhones.Purchase(ctx, decodeContractRequest[cloudphone.PurchaseRequest](t, fixture.Request))
	case "CloudPhones.Renew":
		return client.CloudPhones.Renew(ctx, decodeContractRequest[cloudphone.RenewRequest](t, fixture.Request))
	case "CloudPhones.Share":
		return client.CloudPhones.Share(ctx, decodeContractRequest[cloudphone.ShareRequest](t, fixture.Request))
	case "CloudPhones.InitProxy":
		return client.CloudPhones.InitProxy(ctx, decodeContractRequest[cloudphone.InitProxyRequest](t, fixture.Request))
	case "CloudPhones.SetProxyEnabled":
		req := decodeContractRequest[cloudphone.SwitchRequest](t, fixture.Request)
		return client.CloudPhones.SetProxyEnabled(ctx, req.ImageIDs, req.Status == 1)
	case "CloudPhones.Timezones":
		req := decodeContractRequest[cloudphone.TimezoneListRequest](t, fixture.Request)
		return client.CloudPhones.Timezones(ctx, req.ISO)
	case "CloudPhones.Languages":
		return client.CloudPhones.Languages(ctx)
	case "CloudPhones.ResetDevice":
		return client.CloudPhones.ResetDevice(ctx, decodeContractRequest[cloudphone.ResetDeviceRequest](t, fixture.Request))
	case "CloudPhones.Live":
		return client.CloudPhones.Live(ctx, decodeContractRequest[cloudphone.LiveRequest](t, fixture.Request))
	case "CloudPhones.ScanCode":
		req := decodeContractRequest[cloudphone.ScanRequest](t, fixture.Request)
		return client.CloudPhones.ScanCode(ctx, req.ImageID, req.ID)
	case "CloudPhones.SetAutoRenewal":
		req := decodeContractRequest[cloudphone.SwitchRequest](t, fixture.Request)
		return client.CloudPhones.SetAutoRenewal(ctx, req.ImageIDs, req.Status == 1)
	case "CloudPhones.ConvertVersion":
		return client.CloudPhones.ConvertVersion(ctx, decodeContractRequest[cloudphone.ConvertVersionRequest](t, fixture.Request))
	case "CloudNumbers.List":
		return client.CloudNumbers.List(ctx, decodeContractRequest[cloudnumber.ListRequest](t, fixture.Request))
	case "CloudNumbers.SMSList":
		return client.CloudNumbers.SMSList(ctx, decodeContractRequest[cloudnumber.SMSListRequest](t, fixture.Request))
	case "CloudNumbers.WriteSMS":
		return client.CloudNumbers.WriteSMS(ctx, decodeContractRequest[cloudnumber.WriteSMSRequest](t, fixture.Request))
	case "CloudNumbers.PurchasePackages":
		return client.CloudNumbers.PurchasePackages(ctx)
	case "CloudNumbers.PackageDurations":
		return client.CloudNumbers.PackageDurations(ctx, decodeContractRequest[cloudnumber.PackageDurationsRequest](t, fixture.Request))
	case "CloudNumbers.Purchase":
		return client.CloudNumbers.Purchase(ctx, decodeContractRequest[cloudnumber.PurchaseRequest](t, fixture.Request))
	case "CloudNumbers.RenewalPackages":
		req := decodeContractRequest[cloudnumber.RenewalPackagesRequest](t, fixture.Request)
		return client.CloudNumbers.RenewalPackages(ctx, req.NumberIDs)
	case "CloudNumbers.Renew":
		return client.CloudNumbers.Renew(ctx, decodeContractRequest[cloudnumber.RenewRequest](t, fixture.Request))
	case "CloudNumbers.SetAutoRenewal":
		req := decodeContractRequest[cloudnumber.AutoRenewalRequest](t, fixture.Request)
		return client.CloudNumbers.SetAutoRenewal(ctx, req.NumberIDs, req.Type == 1)
	case "Groups.List":
		return client.Groups.List(ctx, decodeContractRequest[group.ListRequest](t, fixture.Request))
	case "Groups.AddPhones":
		req := decodeContractRequest[group.GroupImagesRequest](t, fixture.Request)
		return client.Groups.AddPhones(ctx, req.ID, req.ImageIDs)
	case "Groups.MovePhones":
		req := decodeContractRequest[group.GroupImagesRequest](t, fixture.Request)
		return client.Groups.MovePhones(ctx, req.ID, req.ImageIDs)
	case "Groups.Create":
		return client.Groups.Create(ctx, decodeContractRequest[group.CreateRequest](t, fixture.Request))
	case "Groups.Update":
		return client.Groups.Update(ctx, decodeContractRequest[group.UpdateRequest](t, fixture.Request))
	case "Groups.Delete":
		req := decodeContractRequest[group.IDsRequest](t, fixture.Request)
		return client.Groups.Delete(ctx, req.IDs)
	case "Proxies.List":
		return client.Proxies.List(ctx, decodeContractRequest[proxy.ListRequest](t, fixture.Request))
	case "Proxies.Add":
		return client.Proxies.Add(ctx, decodeContractRequest[proxy.AddRequest](t, fixture.Request))
	case "Proxies.Delete":
		req := decodeContractRequest[proxy.IDsRequest](t, fixture.Request)
		return client.Proxies.Delete(ctx, req.IDs)
	case "Proxies.RefreshURLs":
		req := decodeContractRequest[proxy.IDsRequest](t, fixture.Request)
		return client.Proxies.RefreshURLs(ctx, req.IDs)
	case "Proxies.Update":
		return client.Proxies.Update(ctx, decodeContractRequest[proxy.UpdateRequest](t, fixture.Request))
	case "Proxies.Check":
		return client.Proxies.Check(ctx, decodeContractRequest[proxy.CheckRequest](t, fixture.Request))
	case "Proxies.SetAutoRenewal":
		req := decodeContractRequest[proxy.AutoRenewalRequest](t, fixture.Request)
		return client.Proxies.SetAutoRenewal(ctx, req.ProxyIDs, req.Type == 1)
	case "SubscriptionStartups.List":
		return client.SubscriptionStartups.List(ctx, decodeContractRequest[subscriptionstartup.ListRequest](t, fixture.Request))
	case "SubscriptionStartups.Purchase":
		return client.SubscriptionStartups.Purchase(ctx, decodeContractRequest[subscriptionstartup.PurchaseRequest](t, fixture.Request))
	case "SubscriptionStartups.Renew":
		return client.SubscriptionStartups.Renew(ctx, decodeContractRequest[subscriptionstartup.RenewRequest](t, fixture.Request))
	case "SubscriptionStartups.SetAutoRenewal":
		req := decodeContractRequest[subscriptionstartup.AutoRenewalRequest](t, fixture.Request)
		return client.SubscriptionStartups.SetAutoRenewal(ctx, req.PhoneIDs, req.Status == 1)
	case "Apps.ListPlatform":
		return client.Apps.ListPlatform(ctx, decodeContractRequest[app.ListRequest](t, fixture.Request))
	case "Apps.ListTeam":
		return client.Apps.ListTeam(ctx, decodeContractRequest[app.ListRequest](t, fixture.Request))
	case "Apps.Install":
		return client.Apps.Install(ctx, decodeContractRequest[app.InstallRequest](t, fixture.Request))
	case "Apps.Installed":
		req := decodeContractRequest[app.InstalledRequest](t, fixture.Request)
		return client.Apps.Installed(ctx, req.ImageID)
	case "Apps.Uninstall":
		req := decodeContractRequest[app.PackageOperationRequest](t, fixture.Request)
		return client.Apps.Uninstall(ctx, req.ImageIDs, req.Pkg)
	case "Apps.Start":
		req := decodeContractRequest[app.PackageOperationRequest](t, fixture.Request)
		return client.Apps.Start(ctx, req.ImageIDs, req.Pkg)
	case "Apps.Stop":
		req := decodeContractRequest[app.PackageOperationRequest](t, fixture.Request)
		return client.Apps.Stop(ctx, req.ImageIDs, req.Pkg)
	case "CloudDisk.List":
		return client.CloudDisk.List(ctx, decodeContractRequest[clouddisk.ListRequest](t, fixture.Request))
	case "CloudDisk.PushFiles":
		return client.CloudDisk.PushFiles(ctx, decodeContractRequest[clouddisk.PushFilesRequest](t, fixture.Request))
	case "CloudDisk.Delete":
		req := decodeContractRequest[clouddisk.DeleteRequest](t, fixture.Request)
		return client.CloudDisk.Delete(ctx, req.IDs)
	case "CloudDisk.SignedURL":
		return client.CloudDisk.SignedURL(ctx, decodeContractRequest[clouddisk.SignedURLRequest](t, fixture.Request))
	case "Automation.ListUserTemplates":
		return client.Automation.ListUserTemplates(ctx, decodeContractRequest[automation.TemplateListRequest](t, fixture.Request))
	case "Automation.ListOfficialTemplates":
		return client.Automation.ListOfficialTemplates(ctx, decodeContractRequest[automation.TemplateListRequest](t, fixture.Request))
	case "Automation.UserTemplateInfo":
		req := decodeContractRequest[automation.TemplateInfoRequest](t, fixture.Request)
		return client.Automation.UserTemplateInfo(ctx, req.ID)
	case "Automation.OfficialTemplateInfo":
		req := decodeContractRequest[automation.TemplateInfoRequest](t, fixture.Request)
		return client.Automation.OfficialTemplateInfo(ctx, req.ID)
	case "Automation.CreateTask":
		return client.Automation.CreateTask(ctx, decodeContractRequest[automation.CreateTaskRequest](t, fixture.Request))
	case "Automation.CreatePlan":
		return client.Automation.CreatePlan(ctx, decodeContractRequest[automation.CreateTaskRequest](t, fixture.Request))
	case "Automation.ListPlans":
		return client.Automation.ListPlans(ctx, decodeContractRequest[automation.PlanListRequest](t, fixture.Request))
	case "Automation.SavePlan":
		return client.Automation.SavePlan(ctx, decodeContractRequest[automation.SavePlanRequest](t, fixture.Request))
	case "Automation.SetPlanStatus":
		return client.Automation.SetPlanStatus(ctx, decodeContractRequest[automation.SetPlanStatusRequest](t, fixture.Request))
	case "Automation.DeletePlan":
		req := decodeContractRequest[automation.DeletePlanRequest](t, fixture.Request)
		return client.Automation.DeletePlan(ctx, req.ID)
	case "Automation.ListTasks":
		return client.Automation.ListTasks(ctx, decodeContractRequest[automation.TaskListRequest](t, fixture.Request))
	case "Automation.ListTaskLogs":
		return client.Automation.ListTaskLogs(ctx, decodeContractRequest[automation.TaskLogListRequest](t, fixture.Request))
	case "Automation.UpdateTaskTime":
		return client.Automation.UpdateTaskTime(ctx, decodeContractRequest[automation.UpdateTaskTimeRequest](t, fixture.Request))
	case "Automation.SetTaskStatus":
		return client.Automation.SetTaskStatus(ctx, decodeContractRequest[automation.SetTaskStatusRequest](t, fixture.Request))
	case "Team.Orders":
		return client.Team.Orders(ctx, decodeContractRequest[team.OrderListRequest](t, fixture.Request))
	case "Team.Balance":
		return client.Team.Balance(ctx)
	case "Team.Resources":
		return client.Team.Resources(ctx)
	case "CloudPhones.SetADBIPWhitelist":
		req := decodeContractRequest[cloudphone.SetADBIPWhitelistRequest](t, fixture.Request)
		return client.CloudPhones.SetADBIPWhitelist(ctx, req.IPs)
	case "CloudPhones.AppendADBIPWhitelist":
		req := decodeContractRequest[cloudphone.SetADBIPWhitelistRequest](t, fixture.Request)
		return client.CloudPhones.AppendADBIPWhitelist(ctx, req.IPs)
	default:
		t.Fatalf("no public SDK call registered for %s", fixture.Operation)
		return nil, nil
	}
}
