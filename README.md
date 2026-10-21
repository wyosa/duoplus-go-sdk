# DuoPlus Go SDK

English | [简体中文](README.zh-CN.md)

An unofficial Go SDK for the [DuoPlus OpenAPI](https://help.duoplus.net/docs/api-reference).

Good fit for:

- Calling the DuoPlus OpenAPI directly from Go services
- Writing cloud phone maintenance scripts, batch jobs and internal tools
- Integrating proxy, cloud drive and automation features into your own systems

## Feature Overview

Modules covered by the SDK:

- Cloud phone management
- Cloud numbers
- Cloud phone groups
- Proxy management
- Subscription startup
- Application management
- Cloud drive management
- Automation
- Team

### Not covered

All `/api/v1` endpoints from the API reference are implemented. Out of scope:

- **AI-Controlled Cloud Phone API**. It uses a separate gateway (`agent-gateway.duoplus.net`) with Bearer auth
  and a different response format.
- **Advanced Commands** (`DuoPlusDumpUI`, `hideAccb`). These are command strings rather than endpoints, so you can
  run them through `client.CloudPhones.Command(...)`.

## Project Structure

```text
github.com/wyosa/duoplus-go-sdk/
├── app/                    # Application management
├── automation/             # Automation
├── clouddisk/              # Cloud drive management
├── cloudnumber/            # Cloud numbers
├── cloudphone/             # Cloud phone management
├── common/                 # Shared request/response types
├── group/                  # Cloud phone groups
├── internal/clientcore/    # HTTP core: auth, rate limiting, error handling
├── proxy/                  # Proxy management
├── subscriptionstartup/    # Subscription startup
├── team/                   # Team: orders, balance, resources
├── examples/               # Runnable examples
├── client.go               # Root client entry point
├── LICENSE
└── README.md
```

## Installation

Requirements:

- Go 1.22+

```bash
go get github.com/wyosa/duoplus-go-sdk
```

To install an explicit version in your project:

```bash
go get github.com/wyosa/duoplus-go-sdk@latest
```

For long-term production use, pin a tag rather than always tracking the latest commit.

## API Conventions

The SDK follows the conventions in the current public DuoPlus docs:

- Every endpoint uses `POST`
- Request bodies are `application/json`
- The auth header is `DuoPlus-API-Key`
- The language header is `Lang` (`en`, `zh`, `zh-TW`, `ru`)
- The default host is `https://openapi.duoplus.net`
- The mainland China host `https://openapi.duoplus.cn` is available as `duoplus.DefaultCNBaseURL`
- The QPS limit is 1 **per endpoint**, so by default the SDK waits at least 1 second between two calls to the same endpoint

Responses use this envelope:

```json
{
	"code": 200,
	"data": {},
	"message": "Success"
}
```

Where:

- `code = 200` means success
- Any other code is returned as `*duoplus.APIError`

## Security

- **Redirects are never followed.** `net/http` would forward the `DuoPlus-API-Key` header to any redirect target,
  including other hosts. The SDK copies the HTTP client (yours too, if you pass one with `WithHTTPClient`) and
  refuses every redirect. Your own `*http.Client` is not modified.
- **HTTPS is enforced.** `WithBaseURL` accepts only `https://` URLs. Plain `http://` is allowed only for loopback
  hosts (`localhost`, `127.0.0.1`, `::1`), for local testing. URLs with credentials, a query or a fragment are rejected.
- **Response size is capped** at 10 MiB, and `APIError.Body` is truncated to 4 KiB so large or sensitive raw
  bodies don't end up in logs.
- Keep the API key in an environment variable or a secret store. `.env` is git-ignored; see `.env.example`.

## Design Notes

- Default host is `https://openapi.duoplus.net`
- Default language is `en`
- A built-in minimum interval of 1 second per endpoint, as required by the docs
- Every endpoint uses `POST + JSON`
- Business errors are returned as `*duoplus.APIError`
- Shared types live in the `common` package; module request/response types live in their own subpackages
- Optional request fields where `0` or `""` is a meaningful value are pointers (for example
  `RenewalStatus *int`). Use `common.Ptr(0)` to send an explicit zero, or leave the field `nil` to omit it

## Importing Modules

The recommended import pattern:

```go
import (
	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
	"github.com/wyosa/duoplus-go-sdk/common"
)
```

Conventions:

- The root `duoplus` package creates the client and takes global options
- Subpackages such as `cloudphone`, `proxy` and `automation` hold each module's request and response types
- `common` holds pagination, message responses and other shared types

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
	"github.com/wyosa/duoplus-go-sdk/common"
)

func main() {
	apiKey := os.Getenv("DUOPLUS_API_KEY")
	if apiKey == "" {
		log.Fatal("DUOPLUS_API_KEY is required")
	}

	client, err := duoplus.NewClient(
		apiKey,
		duoplus.WithBaseURL(duoplus.DefaultBaseURL),
		duoplus.WithLanguage("en"),
	)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.CloudPhones.List(context.Background(), cloudphone.ListRequest{
		PaginationRequest: common.PaginationRequest{Page: 1, PageSize: 10},
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range resp.List {
		fmt.Println(item.ID, item.Name, item.Status)
	}
}
```

Run a ready-made example:

```bash
export DUOPLUS_API_KEY=your-api-key
go run ./examples/basic
```

## Usage

### 1. Create a client

```go
client, err := duoplus.NewClient(
	apiKey,
	duoplus.WithBaseURL(duoplus.DefaultBaseURL),
	duoplus.WithLanguage("en"),
)
```

Options:

- `duoplus.WithBaseURL(...)`
- `duoplus.WithLanguage(...)`
- `duoplus.WithHTTPClient(...)`
- `duoplus.WithMinInterval(...)`

For the mainland China environment:

```go
client, err := duoplus.NewClient(
	apiKey,
	duoplus.WithBaseURL(duoplus.DefaultCNBaseURL),
	duoplus.WithLanguage("zh"),
)
```

### 2. Access features through module services

The root client exposes one service per module:

- `client.CloudPhones`
- `client.CloudNumbers`
- `client.Groups`
- `client.Proxies`
- `client.SubscriptionStartups`
- `client.Apps`
- `client.CloudDisk`
- `client.Automation`
- `client.Team`

### 3. Import request/response types from the module package

For example, listing cloud phones uses:

- `cloudphone.ListRequest`
- `cloudphone.ListResponse`

Pagination and shared response types come from:

- `common.PaginationRequest`
- `common.Pagination`
- `common.MessageResponse`

### 4. Tell read-only calls apart from calls with side effects

These calls have real business impact and should be audited separately when you integrate them:

- Power on, power off, restart
- Pushing files to a cloud phone
- Installing, uninstalling, starting and stopping apps
- Creating, editing and re-running automation tasks
- Purchases and renewals

That is why the business script examples in this repository run as a dry run by default and only make real changes
when `DUOPLUS_EXECUTE=1` is set.

## Common Workflows

### Workflow 1: List cloud phones

1. Create the client
2. Call `client.CloudPhones.List(...)`
3. Iterate over `resp.List`

### Workflow 2: Power on and wait until it's done

1. Call `client.CloudPhones.PowerOn(...)`
2. Poll `client.CloudPhones.Status(...)`
3. Wait until the phone reports it is powered on

See `examples/power-on-wait`.

### Workflow 3: Run a health-check ADB command

1. Make sure the target cloud phone is powered on
2. Call `client.CloudPhones.Command(...)`
3. Check the output for an expected keyword

See `examples/power-on-adb-check`.

### Workflow 4: Push a cloud drive file to a cloud phone

1. Call `client.CloudDisk.List(...)` to find the file
2. Make sure the cloud phone is available
3. Call `client.CloudDisk.PushFiles(...)`

See `examples/real-workflow`.

## Feature List

### Cloud phone management

- List
- Power on, power off, restart
- Status
- Details
- Batch parameter changes (including GPS, SIM with a cloud number, base station)
- Proxy initialization (`InitProxy`), enabling/disabling the proxy (`SetProxyEnabled`)
- Timezone and language lists (`Timezones`, `Languages`)
- Phone model list
- Reset and regenerate device (`ResetDevice`)
- Root permission settings
- ADB command execution
- Enabling/disabling ADB, ADB IP whitelist (`SetADBIPWhitelist` overwrites, `AppendADBIPWhitelist` appends)
- Sharing and share password management
- Tags, connected members, resource list, resolutions
- Live streaming (`Live`), scan code (`ScanCode`)
- Purchase, renewal and auto-renewal (`SetAutoRenewal`)
- Android version conversion (`ConvertVersion`)

Package: `cloudphone`

### Cloud numbers

- Cloud number list
- Cloud number SMS
- Writing SMS to cloud phones
- Purchase package details, package durations (`PurchasePackages`, `PackageDurations`)
- Purchase (`Purchase`)
- Renewal package details and renewal (`RenewalPackages`, `Renew`)
- Auto-renewal (`SetAutoRenewal`)

Package: `cloudnumber`

### Cloud phone groups

- Group list
- Add to group
- Move to group
- Create groups
- Edit groups
- Delete groups

Package: `group`

### Proxy management

- Proxy list
- Batch add
- Batch delete
- Batch refresh URL
- Modify proxy
- Check proxy (`Check`; the call succeeds even when the check fails, so inspect `Success` and `Tips`)
- Auto-renewal (`SetAutoRenewal`)

Package: `proxy`

### Subscription startup

- List
- Purchase
- Renewal
- Auto-renewal (`SetAutoRenewal`)

Package: `subscriptionstartup`

### Application management

- Platform app list
- Team app list
- Batch install
- Installed app list
- Batch uninstall
- Batch start
- Batch close

Package: `app`

### Cloud drive management

- File list
- Push files to a cloud phone
- Upload local files (`UploadFile`, `Upload`, or the raw `SignedURL` step)
- Delete files (`Delete`)

Package: `clouddisk`

### Automation

- Custom template list and details (`UserTemplateInfo`)
- Official template list and details (`OfficialTemplateInfo`)
- Create scheduled task
- Create loop task
- Loop task list
- Edit loop task
- Pause/resume a loop task
- Delete loop task
- Scheduled task list
- Task logs
- Change task publish time
- Cancel/re-run a task

Package: `automation`

### Team

- Order list (`Orders`)
- Balance (`Balance`)
- Team resources (`Resources`)

Package: `team`

## Usage Examples

### Example 1: Check cloud phone status

```go
status, err := client.CloudPhones.Status(ctx, []string{"image-id-1", "image-id-2"})
if err != nil {
	return err
}

for _, item := range status.List {
	fmt.Println(item.ID, item.Status)
}
```

### Example 2: Batch add proxies

```go
resp, err := client.Proxies.Add(ctx, proxy.AddRequest{
	ProxyList: []proxy.Input{
		{
			Protocol: "socks5",
			Host:     "127.0.0.1",
			Port:     3000,
			User:     "user",
			Password: "pass",
			Name:     "proxy-1",
		},
	},
	IPScanChannel: "ip2location",
})
if err != nil {
	return err
}

fmt.Println(resp.Success)
```

### Example 3: Find a cloud drive file and push it to a cloud phone

```go
files, err := client.CloudDisk.List(ctx, clouddisk.ListRequest{
	Keyword:           "apk",
	PaginationRequest: common.PaginationRequest{Page: 1, PageSize: 20},
})
if err != nil {
	return err
}

if len(files.List) > 0 {
	_, err = client.CloudDisk.PushFiles(ctx, clouddisk.PushFilesRequest{
		IDs:      []string{files.List[0].ID},
		ImageIDs: []string{"image-id"},
		DestDir:  "/sdcard/Download",
	})
	if err != nil {
		return err
	}
}
```

### Example 4: Create a scheduled automation task

```go
_, err := client.Automation.CreateTask(ctx, automation.CreateTaskRequest{
	TemplateID:   "template-id",
	TemplateType: 2,
	Name:         "task-name",
	Remark:       "remark",
	Images: []automation.TaskImage{
		{
			ImageID: "image-id",
			IssueAt: "2026-04-01 10:00",
			Config: map[string]automation.ConfigValue{
				"text": {
					Key:      "text",
					Value:    "hello\nworld",
					Type:     "textarea",
					Required: true,
				},
			},
		},
	},
})
if err != nil {
	return err
}
```

### Example 5: Buy a cloud phone without auto-renewal

```go
order, err := client.CloudPhones.Purchase(ctx, cloudphone.PurchaseRequest{
	OS:            "12A",
	Quantity:      1,
	Duration:      "30",
	RenewalStatus: common.Ptr(0), // the server defaults to 1 (auto-renewal on)
})
if err != nil {
	return err
}

fmt.Println(order.OrderID, order.ProductIDs)
```

### Example 6: Upload an APK to the cloud drive

```go
file, err := client.CloudDisk.UploadFile(ctx, "./app.apk", clouddisk.SignedURLRequest{
	IsApp: 1,
	Pkg:   "com.example.app",
})
if err != nil {
	return err
}

fmt.Println(file.Name)
```

The upload goes straight to the presigned storage URL without the API key. The HTTP client timeout does not apply
to it, so bound large uploads with `ctx`.

## Examples Directory

- `examples/basic`: basic cloud phone list
- `examples/automation`: automation template list
- `examples/group-disk`: cloud phone groups and cloud drive
- `examples/power-on-wait`: a real business script that powers on a phone and polls its status until it's done
- `examples/power-on-adb-check`: a real business script that powers on a phone, runs an ADB command and validates the output
- `examples/real-workflow`: a complete workflow that picks a real cloud phone and cloud drive file automatically

How to run:

```bash
export DUOPLUS_API_KEY=your-api-key
go run ./examples/basic
go run ./examples/automation
go run ./examples/group-disk
```

### Example environment variables

| Variable | Purpose | Examples |
| --- | --- | --- |
| `DUOPLUS_API_KEY` | DuoPlus API key | All examples |
| `DUOPLUS_EXECUTE` | Set to `1` to make real changes | `power-on-wait` `power-on-adb-check` `real-workflow` |
| `DUOPLUS_TARGET_IMAGE_ID` | Target cloud phone ID | `power-on-wait` `power-on-adb-check` |
| `DUOPLUS_ADB_COMMAND` | Custom ADB shell command | `power-on-adb-check` |
| `DUOPLUS_EXPECT_SUBSTRING` | Text the ADB output must contain | `power-on-adb-check` |
| `DUOPLUS_DEST_DIR` | Custom push destination | `real-workflow` |

### Real business script examples

#### Script 1: Power on and poll until it's done

Dry run by default; it doesn't actually power anything on:

```bash
export DUOPLUS_API_KEY=your-api-key
go run ./examples/power-on-wait
```

Power on for real and poll the status:

```bash
export DUOPLUS_API_KEY=your-api-key
export DUOPLUS_EXECUTE=1
go run ./examples/power-on-wait
```

Optionally pick the target cloud phone:

```bash
export DUOPLUS_TARGET_IMAGE_ID=your-image-id
```

#### Script 2: Power on, run an ADB command and check the result

Also a dry run by default:

```bash
export DUOPLUS_API_KEY=your-api-key
go run ./examples/power-on-adb-check
```

Run the real flow:

```bash
export DUOPLUS_API_KEY=your-api-key
export DUOPLUS_EXECUTE=1
go run ./examples/power-on-adb-check
```

Optional environment variables:

```bash
export DUOPLUS_TARGET_IMAGE_ID=your-image-id
export DUOPLUS_ADB_COMMAND='getprop ro.product.model'
export DUOPLUS_EXPECT_SUBSTRING='Pixel'
```

This script:

- Picks a cloud phone automatically, preferring one that is powered off
- Powers it on if needed and polls until it is on
- Runs one ADB shell command
- Does a simple check on the output, which is useful as a pre-flight health check

#### Script 3: A more complete business workflow

This script:

- Picks a cloud phone automatically
- Picks a file from the cloud drive automatically
- Powers the phone on first if it is off, and polls until it is on
- Pushes the file to the destination directory on the phone

Also a dry run by default:

```bash
export DUOPLUS_API_KEY=your-api-key
go run ./examples/real-workflow
```

Run the real flow:

```bash
export DUOPLUS_API_KEY=your-api-key
export DUOPLUS_EXECUTE=1
go run ./examples/real-workflow
```

Optionally set the push destination:

```bash
export DUOPLUS_DEST_DIR=/sdcard/Download
```

Notes:

- A real run may power on a cloud phone and incur charges
- The scripts default to a dry run precisely to avoid mistakes
- The script prefers a powered-off phone and the first cloud drive file it finds

## Error Handling

When the API returns a business error, you get a `*duoplus.APIError`.

```go
resp, err := client.CloudPhones.List(ctx, cloudphone.ListRequest{})
if err != nil {
	var apiErr *duoplus.APIError
	if errors.As(err, &apiErr) {
		fmt.Println(apiErr.Code, apiErr.Message)
	}
	return err
}

_ = resp
```

Recommendations:

- Use `errors.As(err, &apiErr)` first to check for a business error
- Read-only endpoints can be retried a limited number of times, but respect the 1 QPS limit
- For power-on, push and purchase calls, log the target IDs and the response for troubleshooting and auditing
- Batch operations report per-item failures in `Fail` (and `FailReason` where available) even when the call itself succeeds

## Docs Used

The SDK follows the **English** API reference (https://help.duoplus.net/docs/api-reference). Where it differs from
the Chinese docs, the English version wins:

| Endpoint | English docs (used) | Chinese docs |
| --- | --- | --- |
| Details `/cloudPhone/info` | `image_ids` (string) | `image_id` |
| Batch Set Root `/cloudPhone/batchRoot` | `pkgs` | `packages` |
| Write SMS `/cloudNumber/imageWriteSms` | `image_id` is an array | `image_id` is a string |

The one exception is cloud number auto-renewal: the English page prints the proxy path `/api/v1/proxy/autoRenewal`
by mistake, so the SDK uses `/api/v1/cloudNumber/autoRenewal` from the Chinese docs.

Inconsistencies within the docs themselves:

- Edit Loop Task (`/automation/savePlan`): the example sends `weeks`/`days` as numbers, while Create Loop Task sends strings. The SDK sends strings.
- `duration` in purchase/renewal requests is required, but the server defaults to 30 days, so the SDK omits it when empty.
- Where a table and an example disagree on a type (for example a template `id` typed as `int` but shown as `"mgRg1"`), the SDK uses the type the example and the other endpoints use.

## Release Recommendations

If you plan to maintain this SDK publicly over the long term:

- Use semantic versioning, e.g. `v0.1.0`, `v0.2.0`
- Bump the major version, or at least call it out in the README, on breaking changes
- When adding endpoints, update the "Feature List" section of the README
- Keep `examples/` runnable so the docs and the code don't drift apart

## Notes

- This is an unofficial SDK built from the public DuoPlus docs
- If DuoPlus changes fields, paths or the meaning of business codes, the SDK needs to be updated too
- Before going to production, verify the key workflows against test resources

## License

[MIT](LICENSE)
