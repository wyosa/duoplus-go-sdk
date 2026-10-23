package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/examples/internal/phoneutil"
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

	phone, err := phoneutil.Select(ctx, client, targetID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("selected image_id=%s name=%s status=%d\n", phone.ID, phone.Name, phone.Status)
	if !execute {
		fmt.Println("dry-run only: set DUOPLUS_EXECUTE=1 to actually power on and wait")
		return
	}

	if err := phoneutil.EnsureOn(ctx, client, phone, 2*time.Minute, 3*time.Second); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("image_id=%s is powered on\n", phone.ID)
}
