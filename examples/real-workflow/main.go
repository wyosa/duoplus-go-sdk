// Command real-workflow powers on a cloud phone and pushes a cloud drive file to it.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/clouddisk"
	"github.com/wyosa/duoplus-go-sdk/common"
	"github.com/wyosa/duoplus-go-sdk/examples/internal/phoneutil"
)

const defaultDestDir = "/sdcard/Download"

func main() {
	apiKey := os.Getenv("DUOPLUS_API_KEY")
	if apiKey == "" {
		log.Fatal("DUOPLUS_API_KEY is required")
	}

	client, err := duoplus.NewClient(apiKey)
	if err != nil {
		log.Fatal(err)
	}

	if err := run(context.Background(), client); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, client *duoplus.Client) error {
	execute := os.Getenv("DUOPLUS_EXECUTE") == "1"
	destDir := os.Getenv("DUOPLUS_DEST_DIR")
	if destDir == "" {
		destDir = defaultDestDir
	}

	phone, err := phoneutil.Select(ctx, client, os.Getenv("DUOPLUS_TARGET_IMAGE_ID"))
	if err != nil {
		return err
	}

	file, err := pickDiskFile(ctx, client)
	if err != nil {
		return err
	}

	fmt.Printf("workflow target phone: id=%s name=%s status=%d\n", phone.ID, phone.Name, phone.Status)
	fmt.Printf("workflow target file: id=%s name=%s\n", file.ID, file.Name)
	fmt.Printf("push destination: %s\n", destDir)

	if !execute {
		fmt.Println("dry-run only: set DUOPLUS_EXECUTE=1 to power on if needed and push the file")
		return nil
	}

	if err := phoneutil.EnsureOn(ctx, client, phone, 2*time.Minute, 3*time.Second); err != nil {
		return err
	}

	pushResp, err := client.CloudDisk.PushFiles(ctx, clouddisk.PushFilesRequest{
		IDs:      []string{file.ID},
		ImageIDs: []string{phone.ID},
		DestDir:  destDir,
	})
	if err != nil {
		return err
	}

	fmt.Printf("push result message=%s success=%d fail=%d\n", pushResp.Message, len(pushResp.Success), len(pushResp.Fail))
	matched := false
	for _, item := range pushResp.Success {
		fmt.Printf("push success image_id=%s file_id=%s\n", item.ImageID, item.ID)
		if item.ImageID == phone.ID && item.ID == file.ID {
			matched = true
		}
	}
	for _, item := range pushResp.Fail {
		fmt.Printf("push fail image_id=%s file_id=%s err=%s\n", item.ImageID, item.ID, item.Err)
	}
	if len(pushResp.Fail) > 0 {
		return fmt.Errorf("file push failed for %d item(s): %s", len(pushResp.Fail), pushResp.Fail[0].Err)
	}
	if !matched {
		return fmt.Errorf("file push was not confirmed for image_id=%s file_id=%s", phone.ID, file.ID)
	}
	return nil
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
