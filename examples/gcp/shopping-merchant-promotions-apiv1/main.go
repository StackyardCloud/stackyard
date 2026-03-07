package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	promotions "cloud.google.com/go/shopping/merchant/promotions/apiv1"
	"cloud.google.com/go/shopping/merchant/promotions/apiv1/promotionspb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	intervalpb "google.golang.org/genproto/googleapis/type/interval"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	accountID := getenv("STACKYARD_GCP_MERCHANT_PROMOTIONS_ACCOUNT_ID", "123456")
	dataSourceID := getenv("STACKYARD_GCP_MERCHANT_PROMOTIONS_DATASOURCE_ID", "104628")
	promotionID := getenv("STACKYARD_GCP_MERCHANT_PROMOTIONS_ID", "promo-1001")
	contentLanguage := strings.ToLower(getenv("STACKYARD_GCP_MERCHANT_PROMOTIONS_CONTENT_LANGUAGE", "en"))
	targetCountry := strings.ToUpper(getenv("STACKYARD_GCP_MERCHANT_PROMOTIONS_TARGET_COUNTRY", "US"))

	parent := fmt.Sprintf("accounts/%s", accountID)
	dataSource := fmt.Sprintf("%s/dataSources/%s", parent, dataSourceID)
	promotionToken := fmt.Sprintf("%s~%s~%s", contentLanguage, targetCountry, promotionID)
	promotionName := fmt.Sprintf("%s/promotions/%s", parent, promotionToken)

	fmt.Printf("Stackyard GCP Shopping Merchant Promotions shopping/merchant/promotions/apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-promotions-apiv1",
		},
	}

	client, err := promotions.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant promotions client: %v", err)
	}
	defer closeClient("merchant promotions", client.Close)

	insertResp, err := client.InsertPromotion(ctx, &promotionspb.InsertPromotionRequest{
		Parent:     parent,
		DataSource: dataSource,
		Promotion: &promotionspb.Promotion{
			PromotionId:       promotionID,
			ContentLanguage:   contentLanguage,
			TargetCountry:     targetCountry,
			RedemptionChannel: []promotionspb.RedemptionChannel{promotionspb.RedemptionChannel_ONLINE},
			Attributes: &promotionspb.Attributes{
				ProductApplicability: promotionspb.ProductApplicability_ALL_PRODUCTS,
				OfferType:            promotionspb.OfferType_NO_CODE,
				LongTitle:            "Stackyard Promotion " + promotionID,
				CouponValueType:      promotionspb.CouponValueType_MONEY_OFF,
				PromotionEffectiveTimePeriod: &intervalpb.Interval{
					StartTime: timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
					EndTime:   timestamppb.New(time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC)),
				},
			},
		},
	})
	if err != nil {
		exitf("InsertPromotion failed: %v", err)
	}
	if strings.TrimSpace(insertResp.GetName()) == "" {
		exitf("InsertPromotion returned empty name")
	}
	fmt.Printf("InsertPromotion succeeded: %s\n", insertResp.GetName())

	getResp, err := client.GetPromotion(ctx, &promotionspb.GetPromotionRequest{Name: promotionName})
	if err != nil {
		exitf("GetPromotion failed: %v", err)
	}
	if strings.TrimSpace(getResp.GetName()) == "" {
		exitf("GetPromotion returned empty name")
	}
	fmt.Printf("GetPromotion succeeded: %s\n", getResp.GetName())

	it := client.ListPromotions(ctx, &promotionspb.ListPromotionsRequest{
		Parent:   parent,
		PageSize: 1,
	})
	first, err := it.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		exitf("ListPromotions failed: %v", err)
	}
	if err == nil && first != nil {
		fmt.Printf("ListPromotions succeeded: first=%s\n", first.GetName())
	} else {
		fmt.Println("ListPromotions succeeded: no promotions returned")
	}

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, endpoint string) error {
	target := strings.TrimRight(endpoint, "/") + "/v1/projects/stackyard/locations/us-central1/shopping_merchant_promotions/sample?stackyard_contract_probe=1"
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
