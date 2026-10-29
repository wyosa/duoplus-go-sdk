# DuoPlus Go SDK

[DuoPlus API reference](https://help.duoplus.net/docs/api-reference)

## Requirements

- Go 1.22+
- DuoPlus API key in `DUOPLUS_API_KEY`

```sh
go get github.com/wyosa/duoplus-go-sdk
```

## Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
)

func main() {
	client, err := duoplus.NewClient(os.Getenv("DUOPLUS_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	phones, err := client.CloudPhones.List(ctx, cloudphone.ListRequest{})
	if err != nil {
		log.Fatal(err)
	}
	for _, phone := range phones.List {
		fmt.Println(phone.ID, phone.Name, phone.Status)
	}
}
```

```sh
export DUOPLUS_API_KEY=your-api-key
go run ./examples/basic
```

| Example | Operation |
| --- | --- |
| [basic](examples/basic/main.go) | List cloud phones |
| [automation](examples/automation/main.go) | List automation templates |
| [group-disk](examples/group-disk/main.go) | List groups and cloud drive files |
| [power-on-wait](examples/power-on-wait/main.go) | Power on and wait |
| [power-on-adb-check](examples/power-on-adb-check/main.go) | Power on and run an ADB command |
| [real-workflow](examples/real-workflow/main.go) | Power on and push a file |

The last three examples require `DUOPLUS_EXECUTE=1` to make changes.
Use `DUOPLUS_TARGET_IMAGE_ID` to select a phone in the power-on examples.
Temporary startup is billed until the phone is powered off; the examples do not power phones off.
`power-on-adb-check` also reads `DUOPLUS_ADB_COMMAND` and `DUOPLUS_EXPECT_SUBSTRING`; `real-workflow` reads `DUOPLUS_DEST_DIR`.

## API coverage

84 `/api/v1` endpoints, checked on 2026-10-23.

| Service | Endpoints | Operations |
| --- | ---: | --- |
| `CloudPhones` | 30 | Lifecycle, configuration, ADB, sharing, purchase, renewal |
| `CloudNumbers` | 9 | Numbers, SMS, packages, purchase, renewal |
| `Groups` | 6 | List, create, update, delete, add/move phones |
| `Proxies` | 7 | List, add, update, delete, refresh, check, auto-renewal |
| `SubscriptionStartups` | 4 | List, purchase, renewal, auto-renewal |
| `Apps` | 7 | Catalogs, install, uninstall, start, stop |
| `CloudDisk` | 4 | List, upload, push, delete |
| `Automation` | 14 | Templates, tasks, loop plans, logs |
| `Team` | 3 | Orders, balance, resources |

The separate AI Cloud Phone gateway is not included.
