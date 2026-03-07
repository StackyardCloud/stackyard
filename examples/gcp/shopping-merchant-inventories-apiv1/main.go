package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	inventories "cloud.google.com/go/shopping/merchant/inventories/apiv1"
	"cloud.google.com/go/shopping/merchant/inventories/apiv1/inventoriespb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	accountID := getenv("STACKYARD_GCP_MERCHANT_INVENTORIES_ACCOUNT_ID", "123456")
	productID := getenv("STACKYARD_GCP_MERCHANT_INVENTORIES_PRODUCT_ID", "sku-1001")
	storeCode := getenv("STACKYARD_GCP_MERCHANT_INVENTORIES_STORE_CODE", "store-nyc")
	regionID := getenv("STACKYARD_GCP_MERCHANT_INVENTORIES_REGION", "us-east1")

	parent := fmt.Sprintf("accounts/%s/products/%s", accountID, productID)
	localName := fmt.Sprintf("%s/localInventories/%s", parent, storeCode)
	regionalName := fmt.Sprintf("%s/regionalInventories/%s", parent, regionID)

	fmt.Printf("Stackyard GCP Shopping Merchant Inventories shopping/merchant/inventories/apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, parent); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-inventories-apiv1",
		},
	}

	localClient, err := inventories.NewLocalInventoryRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create local inventory client: %v", err)
	}
	defer closeClient("local inventory", localClient.Close)

	regionalClient, err := inventories.NewRegionalInventoryRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create regional inventory client: %v", err)
	}
	defer closeClient("regional inventory", regionalClient.Close)

	calls := []callSpec{
		{
			name: "InsertLocalInventory",
			call: func(ctx context.Context) error {
				resp, err := localClient.InsertLocalInventory(ctx, &inventoriespb.InsertLocalInventoryRequest{
					Parent: parent,
					LocalInventory: &inventoriespb.LocalInventory{
						StoreCode: storeCode,
						LocalInventoryAttributes: &inventoriespb.LocalInventoryAttributes{
							Quantity: proto.Int64(8),
						},
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(resp.GetName()); name != "" {
					localName = name
				}
				return nil
			},
		},
		{
			name: "ListLocalInventories",
			call: func(ctx context.Context) error {
				it := localClient.ListLocalInventories(ctx, &inventoriespb.ListLocalInventoriesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "DeleteLocalInventory",
			call: func(ctx context.Context) error {
				return localClient.DeleteLocalInventory(ctx, &inventoriespb.DeleteLocalInventoryRequest{Name: localName})
			},
		},
		{
			name: "InsertRegionalInventory",
			call: func(ctx context.Context) error {
				resp, err := regionalClient.InsertRegionalInventory(ctx, &inventoriespb.InsertRegionalInventoryRequest{
					Parent: parent,
					RegionalInventory: &inventoriespb.RegionalInventory{
						Region: regionID,
						RegionalInventoryAttributes: &inventoriespb.RegionalInventoryAttributes{
							Availability: inventoriespb.RegionalInventoryAttributes_REGIONAL_INVENTORY_AVAILABILITY_UNSPECIFIED.Enum(),
						},
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(resp.GetName()); name != "" {
					regionalName = name
				}
				return nil
			},
		},
		{
			name: "ListRegionalInventories",
			call: func(ctx context.Context) error {
				it := regionalClient.ListRegionalInventories(ctx, &inventoriespb.ListRegionalInventoriesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "DeleteRegionalInventory",
			call: func(ctx context.Context) error {
				return regionalClient.DeleteRegionalInventory(ctx, &inventoriespb.DeleteRegionalInventoryRequest{Name: regionalName})
			},
		},
	}

	for _, spec := range calls {
		if err := spec.call(ctx); err != nil {
			exitf("%s failed: %v", spec.name, err)
		}
		fmt.Printf("%s succeeded\n", spec.name)
	}

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, endpoint, parent string) error {
	target := strings.TrimRight(endpoint, "/") + "/inventories/v1/" + parent + "/localInventories?pageSize=1"
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
