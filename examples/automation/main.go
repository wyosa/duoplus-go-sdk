package main

import (
	"context"
	"fmt"
	"log"
	"os"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/automation"
	"github.com/wyosa/duoplus-go-sdk/common"
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

	templates, err := client.Automation.ListOfficialTemplates(context.Background(), automation.TemplateListRequest{
		PaginationRequest: common.PaginationRequest{Page: 1, PageSize: 10},
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range templates.List {
		fmt.Printf("%s %s\n", item.ID, item.Name)
	}
}
