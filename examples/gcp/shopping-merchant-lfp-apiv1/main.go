package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	lfp "cloud.google.com/go/shopping/merchant/lfp/apiv1"
	"cloud.google.com/go/shopping/merchant/lfp/apiv1/lfppb"
	shoppingtypepb "cloud.google.com/go/shopping/type/typepb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	accountID := getenv("STACKYARD_GCP_MERCHANT_LFP_ACCOUNT_ID", "123456")
	targetAccountID := getenv("STACKYARD_GCP_MERCHANT_LFP_TARGET_ACCOUNT_ID", "567890")
	storeCode := getenv("STACKYARD_GCP_MERCHANT_LFP_STORE_CODE", "store-nyc")
	offerID := getenv("STACKYARD_GCP_MERCHANT_LFP_OFFER_ID", "offer-1001")
	regionCode := strings.ToUpper(getenv("STACKYARD_GCP_MERCHANT_LFP_REGION_CODE", "US"))
	contentLanguage := getenv("STACKYARD_GCP_MERCHANT_LFP_CONTENT_LANGUAGE", "en")

	parent := fmt.Sprintf("accounts/%s", accountID)
	storeName := fmt.Sprintf("%s/lfpStores/%s~%s", parent, targetAccountID, storeCode)
	merchantStateName := fmt.Sprintf("%s/lfpMerchantStates/%s", parent, targetAccountID)

	targetAccountInt, err := parsePositiveInt64(targetAccountID)
	if err != nil {
		exitf("invalid STACKYARD_GCP_MERCHANT_LFP_TARGET_ACCOUNT_ID: %v", err)
	}

	fmt.Printf("Stackyard GCP Shopping Merchant LFP shopping/merchant/lfp/apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, parent, targetAccountID); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-lfp-apiv1",
		},
	}

	storeClient, err := lfp.NewLfpStoreRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create lfp store client: %v", err)
	}
	defer closeClient("lfp store", storeClient.Close)

	inventoryClient, err := lfp.NewLfpInventoryRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create lfp inventory client: %v", err)
	}
	defer closeClient("lfp inventory", inventoryClient.Close)

	saleClient, err := lfp.NewLfpSaleRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create lfp sale client: %v", err)
	}
	defer closeClient("lfp sale", saleClient.Close)

	merchantStateClient, err := lfp.NewLfpMerchantStateRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create lfp merchant state client: %v", err)
	}
	defer closeClient("lfp merchant state", merchantStateClient.Close)

	calls := []callSpec{
		{
			name: "InsertLfpStore",
			call: func(ctx context.Context) error {
				resp, err := storeClient.InsertLfpStore(ctx, &lfppb.InsertLfpStoreRequest{
					Parent: parent,
					LfpStore: &lfppb.LfpStore{
						TargetAccount: targetAccountInt,
						StoreCode:     storeCode,
						StoreAddress:  "1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA",
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(resp.GetName()); name != "" {
					storeName = name
				}
				return nil
			},
		},
		{
			name: "GetLfpStore",
			call: func(ctx context.Context) error {
				_, err := storeClient.GetLfpStore(ctx, &lfppb.GetLfpStoreRequest{Name: storeName})
				return err
			},
		},
		{
			name: "ListLfpStores",
			call: func(ctx context.Context) error {
				it := storeClient.ListLfpStores(ctx, &lfppb.ListLfpStoresRequest{
					Parent:        parent,
					TargetAccount: targetAccountInt,
					PageSize:      1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "InsertLfpInventory",
			call: func(ctx context.Context) error {
				_, err := inventoryClient.InsertLfpInventory(ctx, &lfppb.InsertLfpInventoryRequest{
					Parent: parent,
					LfpInventory: &lfppb.LfpInventory{
						TargetAccount:   targetAccountInt,
						StoreCode:       storeCode,
						OfferId:         offerID,
						RegionCode:      regionCode,
						ContentLanguage: contentLanguage,
						Availability:    "in stock",
						Price: &shoppingtypepb.Price{
							CurrencyCode: proto.String("USD"),
							AmountMicros: proto.Int64(12990000),
						},
						Quantity: proto.Int64(7),
					},
				})
				return err
			},
		},
		{
			name: "InsertLfpSale",
			call: func(ctx context.Context) error {
				_, err := saleClient.InsertLfpSale(ctx, &lfppb.InsertLfpSaleRequest{
					Parent: parent,
					LfpSale: &lfppb.LfpSale{
						TargetAccount:   targetAccountInt,
						StoreCode:       storeCode,
						OfferId:         offerID,
						RegionCode:      regionCode,
						ContentLanguage: contentLanguage,
						Gtin:            "00012345678905",
						Price: &shoppingtypepb.Price{
							CurrencyCode: proto.String("USD"),
							AmountMicros: proto.Int64(14990000),
						},
						Quantity: 1,
						SaleTime: timestamppb.New(time.Date(2026, time.January, 1, 12, 34, 56, 0, time.UTC)),
					},
				})
				return err
			},
		},
		{
			name: "GetLfpMerchantState",
			call: func(ctx context.Context) error {
				_, err := merchantStateClient.GetLfpMerchantState(ctx, &lfppb.GetLfpMerchantStateRequest{
					Name: merchantStateName,
				})
				return err
			},
		},
		{
			name: "DeleteLfpStore",
			call: func(ctx context.Context) error {
				return storeClient.DeleteLfpStore(ctx, &lfppb.DeleteLfpStoreRequest{Name: storeName})
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

func waitForStackyardReady(ctx context.Context, endpoint, parent, targetAccount string) error {
	target := strings.TrimRight(endpoint, "/") + "/lfp/v1/" + parent + "/lfpStores?targetAccount=" + targetAccount + "&pageSize=1"
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

func parsePositiveInt64(value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("must be a positive integer")
	}
	return parsed, nil
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
