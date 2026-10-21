package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/clouddisk"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
	"github.com/wyosa/duoplus-go-sdk/common"
)

const (
	statusPoweredOn  = 1
	statusPoweredOff = 2
	defaultDestDir   = "/sdcard/Download"
)

func main() {
	apiKey := os.Getenv("DUOPLUS_API_KEY")
	if apiKey == "" {
		log.Fatal("DUOPLUS_API_KEY is required")
	}

	client, err := duoplus.NewClient(apiKey)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	execute := os.Getenv("DUOPLUS_EXECUTE") == "1"
	destDir := os.Getenv("DUOPLUS_DEST_DIR")
	if destDir == "" {
		destDir = defaultDestDir
	}

	phone, err := pickPhone(ctx, client)
	if err != nil {
		log.Fatal(err)
	}

	file, err := pickDiskFile(ctx, client)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("workflow target phone: id=%s name=%s status=%d\n", phone.ID, phone.Name, phone.Status)
	fmt.Printf("workflow target file: id=%s name=%s\n", file.ID, file.Name)
	fmt.Printf("push destination: %s\n", destDir)

	if !execute {
		fmt.Println("dry-run only: set DUOPLUS_EXECUTE=1 to power on if needed and push the file")
		return
	}

	if phone.Status == statusPoweredOff {
		result, err := client.CloudPhones.PowerOn(ctx, []string{phone.ID})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("power-on request success=%v fail=%v\n", result.Success, result.Fail)
		if err := waitForOn(ctx, client, phone.ID, 2*time.Minute, 3*time.Second); err != nil {
			log.Fatal(err)
		}
	}

	pushResp, err := client.CloudDisk.PushFiles(ctx, clouddisk.PushFilesRequest{
		IDs:      []string{file.ID},
		ImageIDs: []string{phone.ID},
		DestDir:  destDir,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("push result message=%s success=%d fail=%d\n", pushResp.Message, len(pushResp.Success), len(pushResp.Fail))
	for _, item := range pushResp.Success {
		fmt.Printf("push success image_id=%s file_id=%s\n", item.ImageID, item.ID)
	}
	for _, item := range pushResp.Fail {
		fmt.Printf("push fail image_id=%s file_id=%s err=%s\n", item.ImageID, item.ID, item.Err)
	}
}

func pickPhone(ctx context.Context, client *duoplus.Client) (*cloudphone.Phone, error) {
	resp, err := client.CloudPhones.List(ctx, cloudphone.ListRequest{
		PaginationRequest: common.PaginationRequest{Page: 1, PageSize: 20},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.List) == 0 {
		return nil, fmt.Errorf("no cloud phones found")
	}

	for _, item := range resp.List {
		if item.Status == statusPoweredOff {
			return &item, nil
		}
	}

	return &resp.List[0], nil
}

func pickDiskFile(ctx context.Context, client *duoplus.Client) (*clouddisk.File, error) {
	resp, err := client.CloudDisk.List(ctx, clouddisk.ListRequest{
		PaginationRequest: common.PaginationRequest{Page: 1, PageSize: 20},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.List) == 0 {
		return nil, fmt.Errorf("no cloud disk files found")
	}
	return &resp.List[0], nil
}

func waitForOn(ctx context.Context, client *duoplus.Client, imageID string, timeout time.Duration, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		resp, err := client.CloudPhones.Status(ctx, []string{imageID})
		if err != nil {
			return err
		}
		if len(resp.List) == 0 {
			return fmt.Errorf("status response for image_id=%s is empty", imageID)
		}

		current := resp.List[0].Status
		fmt.Printf("poll image_id=%s status=%d\n", imageID, current)
		if current == statusPoweredOn {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for image_id=%s to power on, last status=%d", imageID, current)
		}
		time.Sleep(interval)
	}
}
