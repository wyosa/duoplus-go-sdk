// Package phoneutil contains the cloud phone workflow shared by the examples.
package phoneutil

import (
	"context"
	"fmt"
	"time"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
	"github.com/wyosa/duoplus-go-sdk/common"
)

const (
	statusPoweredOn   = 1
	statusPoweredOff  = 2
	statusPoweringOn  = 10
	statusConfiguring = 11
)

// Select queries an explicit ID directly, or prefers a powered-off phone from
// the first page when no target was requested.
func Select(ctx context.Context, client *duoplus.Client, targetID string) (*cloudphone.Phone, error) {
	req := cloudphone.ListRequest{
		PaginationRequest: common.PaginationRequest{Page: 1, PageSize: 20},
	}
	if targetID != "" {
		req.ImageIDs = []string{targetID}
	}
	resp, err := client.CloudPhones.List(ctx, req)
	if err != nil {
		return nil, err
	}
	if targetID != "" {
		for i := range resp.List {
			if resp.List[i].ID == targetID {
				return &resp.List[i], nil
			}
		}
		return nil, fmt.Errorf("target image_id=%s not found", targetID)
	}
	if len(resp.List) == 0 {
		return nil, fmt.Errorf("no cloud phones found")
	}
	for i := range resp.List {
		if resp.List[i].Status == statusPoweredOff {
			return &resp.List[i], nil
		}
	}
	return &resp.List[0], nil
}

// EnsureOn powers on an off phone, or waits for an ongoing startup. The timeout
// bounds API calls and polling sleeps as well as the power-on request itself.
func EnsureOn(ctx context.Context, client *duoplus.Client, phone *cloudphone.Phone, timeout, interval time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	if interval <= 0 {
		return fmt.Errorf("poll interval must be positive")
	}
	switch phone.Status {
	case statusPoweredOn:
		return nil
	case statusPoweredOff:
		result, err := client.CloudPhones.PowerOn(ctx, []string{phone.ID})
		if err != nil {
			return err
		}
		accepted := false
		for _, id := range result.Success {
			if id == phone.ID {
				accepted = true
			}
		}
		for _, id := range result.Fail {
			if id == phone.ID {
				accepted = false
			}
		}
		if !accepted {
			return fmt.Errorf("power-on request was not accepted for image_id=%s: %s", phone.ID, result.FailReason[phone.ID])
		}
	case statusPoweringOn, statusConfiguring:
		// Startup is already in progress; poll without sending PowerOn again.
	default:
		return fmt.Errorf("image_id=%s cannot be powered on in status=%d", phone.ID, phone.Status)
	}
	return waitForOn(ctx, client, phone.ID, interval)
}

func waitForOn(ctx context.Context, client *duoplus.Client, imageID string, interval time.Duration) error {
	for {
		resp, err := client.CloudPhones.Status(ctx, []string{imageID})
		if err != nil {
			return fmt.Errorf("waiting for image_id=%s to power on: %w", imageID, err)
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("waiting for image_id=%s to power on: %w", imageID, err)
		}
		var current *cloudphone.StatusItem
		for i := range resp.List {
			if resp.List[i].ID == imageID {
				current = &resp.List[i]
				break
			}
		}
		if current == nil {
			return fmt.Errorf("status response does not contain image_id=%s", imageID)
		}
		fmt.Printf("poll image_id=%s status=%d\n", imageID, current.Status)
		switch current.Status {
		case statusPoweredOn:
			return nil
		case statusPoweredOff, statusPoweringOn, statusConfiguring:
			// The asynchronous power-on may still report the old off status.
		default:
			return fmt.Errorf("image_id=%s cannot finish startup in status=%d", imageID, current.Status)
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("waiting for image_id=%s to power on: %w", imageID, ctx.Err())
		case <-timer.C:
		}
	}
}
