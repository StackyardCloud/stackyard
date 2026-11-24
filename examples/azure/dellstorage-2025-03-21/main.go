package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

// Azure Dell Storage example: drives the emulated API through the Azure SDK pipeline.

type sharedKeyPolicy struct{ account string }

func (p *sharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	return req.Next()
}

type step struct {
	name, method, path string
	body               any
	statuses           []int
}

func main() {
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	sub := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	rg := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-dellstorage-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	pipeline := runtime.NewPipeline("stackyard", "dellstorage-2025-03-21", runtime.PipelineOptions{
		PerRetry: []policy.Policy{&sharedKeyPolicy{account: getenv("STACKYARD_AZURE_ACCOUNT", "devstoreaccount1")}},
	}, &policy.ClientOptions{})
	fmt.Printf("Stackyard Azure Dell Storage (dellstorage-2025-03-21) example using %s\n", endpoint)

	group := "/azure/subscriptions/" + sub + "/resourcegroups/" + rg
	base := "/azure/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/"
	armID := strings.TrimPrefix(base, "/azure")
	ok := []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent}
	_ = armID
	q := "?api-version=2025-03-21"
	steps := []step{
		{"CreateResourceGroup", http.MethodPut, group + "?api-version=2021-04-01", map[string]any{"location": region}, ok},
		{"ListOperations", http.MethodGet, "/azure/providers/Dell.Storage/operations" + q, nil, ok},
		{"CreateFileSystem", http.MethodPut, base + "Dell.Storage/filesystems/fs1" + q, map[string]any{"location": region, "properties": map[string]any{"marketplace": map[string]any{"planId": "p", "offerId": "o", "planName": "n"}, "delegatedSubnetCidr": "10.1.0.0/24", "delegatedSubnetId": "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/v/subnets/dell", "user": map[string]any{"email": "ops@example.com"}, "dellReferenceNumber": "DELL-1", "encryption": map[string]any{"encryptionType": "Microsoft-managed keys (MMK)"}}}, ok},
		{"UpdateFileSystem", http.MethodPatch, base + "Dell.Storage/filesystems/fs1" + q, map[string]any{"tags": map[string]any{"env": "dev"}}, ok},
		{"GetFileSystem", http.MethodGet, base + "Dell.Storage/filesystems/fs1" + q, nil, ok},
		{"ListByResourceGroup", http.MethodGet, base + "Dell.Storage/filesystems" + q, nil, ok},
		{"ListBySubscription", http.MethodGet, "/azure/subscriptions/" + sub + "/providers/Dell.Storage/filesystems" + q, nil, ok},
		{"DeleteFileSystem", http.MethodDelete, base + "Dell.Storage/filesystems/fs1" + q, nil, ok},
		{"DeleteResourceGroup", http.MethodDelete, group + "?api-version=2021-04-01", nil, ok},
	}
	for _, s := range steps {
		if err := run(context.Background(), pipeline, endpoint, s); err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", s.name, err)
			os.Exit(1)
		}
	}
}

func run(ctx context.Context, pipeline runtime.Pipeline, endpoint string, s step) error {
	req, err := runtime.NewRequest(ctx, s.method, endpoint+s.path)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if s.body != nil {
		if err := runtime.MarshalAsJSON(req, s.body); err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
	}
	resp, err := pipeline.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	for _, status := range s.statuses {
		if resp.StatusCode == status {
			fmt.Printf("%s -> HTTP %d\n", s.name, resp.StatusCode)
			var pretty any
			if json.Unmarshal(body, &pretty) == nil {
				out, _ := json.MarshalIndent(pretty, "", "  ")
				fmt.Println(string(out))
			}
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
