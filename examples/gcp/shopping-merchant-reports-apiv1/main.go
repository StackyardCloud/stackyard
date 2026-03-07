package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	reports "cloud.google.com/go/shopping/merchant/reports/apiv1"
	"cloud.google.com/go/shopping/merchant/reports/apiv1/reportspb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	accountID := getenv("STACKYARD_GCP_MERCHANT_REPORTS_ACCOUNT_ID", "123456")
	parent := fmt.Sprintf("accounts/%s", accountID)
	query := getenv("STACKYARD_GCP_MERCHANT_REPORTS_QUERY", "SELECT product_view.id, product_view.title FROM product_view")

	fmt.Printf("Stackyard GCP Shopping Merchant Reports shopping/merchant/reports/apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-reports-apiv1",
		},
	}

	client, err := reports.NewReportRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant reports client: %v", err)
	}
	defer closeClient("merchant reports", client.Close)

	it := client.Search(ctx, &reportspb.SearchRequest{
		Parent:   parent,
		Query:    query,
		PageSize: 1,
	})
	first, err := it.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		exitf("Search failed: %v", err)
	}
	if err == nil && first != nil {
		if first.GetProductView() == nil || strings.TrimSpace(first.GetProductView().GetId()) == "" {
			exitf("Search returned row without productView.id")
		}
		fmt.Printf("Search succeeded: first=%s title=%s\n", first.GetProductView().GetId(), first.GetProductView().GetTitle())
	} else {
		fmt.Println("Search succeeded: no rows returned")
	}

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, endpoint string) error {
	target := strings.TrimRight(endpoint, "/") + "/v1/projects/stackyard/locations/us-central1/shopping_merchant_reports/sample?stackyard_contract_probe=1"
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < http.StatusInternalServerError {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for stackyard at %s", target)
}

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	if strings.TrimSpace(t.serviceName) != "" {
		clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	}
	return base.RoundTrip(clone)
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to close %s client: %v\n", name, err)
	}
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
