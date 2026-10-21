package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
	"github.com/wyosa/duoplus-go-sdk/common"
)

const (
	statusPoweredOn  = 1
	statusPoweredOff = 2
	defaultCommand   = "getprop ro.product.model"
	defaultExpect    = ""
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
	targetID := os.Getenv("DUOPLUS_TARGET_IMAGE_ID")
	execute := os.Getenv("DUOPLUS_EXECUTE") == "1"
	command := firstNonEmpty(os.Getenv("DUOPLUS_ADB_COMMAND"), defaultCommand)
	expect := firstNonEmpty(os.Getenv("DUOPLUS_EXPECT_SUBSTRING"), defaultExpect)

	phone, err := selectPhone(ctx, client, targetID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("selected image_id=%s name=%s status=%d\n", phone.ID, phone.Name, phone.Status)
	fmt.Printf("adb command=%q\n", command)
	if expect != "" {
		fmt.Printf("expect substring=%q\n", expect)
	}

	if !execute {
		fmt.Println("dry-run only: set DUOPLUS_EXECUTE=1 to power on if needed, execute ADB command, and validate output")
		return
	}

	if phone.Status != statusPoweredOn {
		result, err := client.CloudPhones.PowerOn(ctx, []string{phone.ID})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("power-on request success=%v fail=%v\n", result.Success, result.Fail)
		if !contains(result.Success, phone.ID) {
			log.Fatalf("power-on request was not accepted for image_id=%s", phone.ID)
		}

		if err := waitForStatus(ctx, client, phone.ID, statusPoweredOn, 2*time.Minute, 3*time.Second); err != nil {
			log.Fatal(err)
		}
	}

	result, err := client.CloudPhones.Command(ctx, phone.ID, command)
	if err != nil {
		log.Fatal(err)
	}

	trimmed := strings.TrimSpace(result.Content)
	fmt.Printf("adb success=%v message=%q\n", result.Success, result.Message)
	fmt.Printf("adb output=%q\n", trimmed)

	if !result.Success {
		log.Fatalf("adb command failed for image_id=%s", phone.ID)
	}

	if expect != "" && !strings.Contains(trimmed, expect) {
		log.Fatalf("adb output validation failed: expected substring %q, got %q", expect, trimmed)
	}

	fmt.Println("adb command validation passed")
}

func selectPhone(ctx context.Context, client *duoplus.Client, targetID string) (*cloudphone.Phone, error) {
	resp, err := client.CloudPhones.List(ctx, cloudphone.ListRequest{
		PaginationRequest: common.PaginationRequest{Page: 1, PageSize: 20},
	})
	if err != nil {
		return nil, err
	}

	if len(resp.List) == 0 {
		return nil, fmt.Errorf("no cloud phones found")
	}

	if targetID != "" {
		for _, item := range resp.List {
			if item.ID == targetID {
				return &item, nil
			}
		}
		return nil, fmt.Errorf("target image_id %s not found on first page", targetID)
	}

	for _, item := range resp.List {
		if item.Status == statusPoweredOff {
			return &item, nil
		}
	}

	return &resp.List[0], nil
}

func waitForStatus(ctx context.Context, client *duoplus.Client, imageID string, want int, timeout time.Duration, interval time.Duration) error {
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
		if current == want {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for image_id=%s to reach status=%d, last status=%d", imageID, want, current)
		}

		time.Sleep(interval)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
