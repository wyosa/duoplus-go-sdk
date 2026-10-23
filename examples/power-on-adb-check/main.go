package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/examples/internal/phoneutil"
)

const (
	defaultCommand = "getprop ro.product.model"
	defaultExpect  = ""
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

	phone, err := phoneutil.Select(ctx, client, targetID)
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

	if err := phoneutil.EnsureOn(ctx, client, phone, 2*time.Minute, 3*time.Second); err != nil {
		log.Fatal(err)
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
