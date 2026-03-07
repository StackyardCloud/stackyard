package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	reviews "cloud.google.com/go/shopping/merchant/reviews/apiv1beta"
	"cloud.google.com/go/shopping/merchant/reviews/apiv1beta/reviewspb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	accountID := getenv("STACKYARD_GCP_MERCHANT_REVIEWS_ACCOUNT_ID", "123456")
	dataSourceID := getenv("STACKYARD_GCP_MERCHANT_REVIEWS_DATASOURCE_ID", "104628")
	merchantReviewID := getenv("STACKYARD_GCP_MERCHANT_REVIEWS_MERCHANT_REVIEW_ID", "merchant-review-1001")
	productReviewID := getenv("STACKYARD_GCP_MERCHANT_REVIEWS_PRODUCT_REVIEW_ID", "product-review-1001")

	parent := fmt.Sprintf("accounts/%s", accountID)
	dataSource := fmt.Sprintf("%s/dataSources/%s", parent, dataSourceID)
	merchantReviewName := fmt.Sprintf("%s/merchantReviews/%s", parent, merchantReviewID)
	productReviewName := fmt.Sprintf("%s/productReviews/%s", parent, productReviewID)

	fmt.Printf("Stackyard GCP Shopping Merchant Reviews shopping/merchant/reviews/apiv1beta clients using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-reviews-apiv1beta",
		},
	}

	merchantClient, err := reviews.NewMerchantReviewsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant reviews client: %v", err)
	}
	defer closeClient("merchant reviews", merchantClient.Close)

	productClient, err := reviews.NewProductReviewsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create product reviews client: %v", err)
	}
	defer closeClient("product reviews", productClient.Close)

	merchantCollectionMethod := reviewspb.MerchantReviewAttributes_AFTER_FULFILLMENT
	merchantInsertResp, err := merchantClient.InsertMerchantReview(ctx, &reviewspb.InsertMerchantReviewRequest{
		Parent:     parent,
		DataSource: dataSource,
		MerchantReview: &reviewspb.MerchantReview{
			MerchantReviewId: merchantReviewID,
			MerchantReviewAttributes: &reviewspb.MerchantReviewAttributes{
				MerchantId:       proto.String("merchant-" + accountID),
				Title:            proto.String("Smooth checkout"),
				Content:          proto.String("Delivery and support were excellent."),
				CollectionMethod: &merchantCollectionMethod,
				ReviewTime:       timestamppb.New(time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)),
			},
		},
	})
	if err != nil {
		exitf("InsertMerchantReview failed: %v", err)
	}
	if strings.TrimSpace(merchantInsertResp.GetName()) == "" {
		exitf("InsertMerchantReview returned empty name")
	}
	fmt.Printf("InsertMerchantReview succeeded: %s\n", merchantInsertResp.GetName())

	merchantGetResp, err := merchantClient.GetMerchantReview(ctx, &reviewspb.GetMerchantReviewRequest{Name: merchantReviewName})
	if err != nil {
		exitf("GetMerchantReview failed: %v", err)
	}
	if strings.TrimSpace(merchantGetResp.GetName()) == "" {
		exitf("GetMerchantReview returned empty name")
	}
	fmt.Printf("GetMerchantReview succeeded: %s\n", merchantGetResp.GetName())

	merchantIterator := merchantClient.ListMerchantReviews(ctx, &reviewspb.ListMerchantReviewsRequest{
		Parent:   parent,
		PageSize: 1,
	})
	merchantFirst, err := merchantIterator.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		exitf("ListMerchantReviews failed: %v", err)
	}
	if err == nil && merchantFirst != nil {
		fmt.Printf("ListMerchantReviews succeeded: first=%s\n", merchantFirst.GetName())
	} else {
		fmt.Println("ListMerchantReviews succeeded: no merchant reviews returned")
	}

	if err := merchantClient.DeleteMerchantReview(ctx, &reviewspb.DeleteMerchantReviewRequest{Name: merchantReviewName}); err != nil {
		exitf("DeleteMerchantReview failed: %v", err)
	}
	fmt.Printf("DeleteMerchantReview succeeded: %s\n", merchantReviewName)

	productInsertResp, err := productClient.InsertProductReview(ctx, &reviewspb.InsertProductReviewRequest{
		Parent:     parent,
		DataSource: dataSource,
		ProductReview: &reviewspb.ProductReview{
			ProductReviewId: productReviewID,
			ProductReviewAttributes: &reviewspb.ProductReviewAttributes{
				Title:            proto.String("Great fit"),
				Content:          proto.String("Comfortable and matches sizing."),
				ReviewTime:       timestamppb.New(time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)),
				CollectionMethod: reviewspb.ProductReviewAttributes_POST_FULFILLMENT,
				ProductNames:     []string{"Stackyard Tee"},
				Skus:             []string{"sku-1001"},
			},
		},
	})
	if err != nil {
		exitf("InsertProductReview failed: %v", err)
	}
	if strings.TrimSpace(productInsertResp.GetName()) == "" {
		exitf("InsertProductReview returned empty name")
	}
	fmt.Printf("InsertProductReview succeeded: %s\n", productInsertResp.GetName())

	productGetResp, err := productClient.GetProductReview(ctx, &reviewspb.GetProductReviewRequest{Name: productReviewName})
	if err != nil {
		exitf("GetProductReview failed: %v", err)
	}
	if strings.TrimSpace(productGetResp.GetName()) == "" {
		exitf("GetProductReview returned empty name")
	}
	fmt.Printf("GetProductReview succeeded: %s\n", productGetResp.GetName())

	productIterator := productClient.ListProductReviews(ctx, &reviewspb.ListProductReviewsRequest{
		Parent:   parent,
		PageSize: 1,
	})
	productFirst, err := productIterator.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		exitf("ListProductReviews failed: %v", err)
	}
	if err == nil && productFirst != nil {
		fmt.Printf("ListProductReviews succeeded: first=%s\n", productFirst.GetName())
	} else {
		fmt.Println("ListProductReviews succeeded: no product reviews returned")
	}

	if err := productClient.DeleteProductReview(ctx, &reviewspb.DeleteProductReviewRequest{Name: productReviewName}); err != nil {
		exitf("DeleteProductReview failed: %v", err)
	}
	fmt.Printf("DeleteProductReview succeeded: %s\n", productReviewName)

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, endpoint string) error {
	target := strings.TrimRight(endpoint, "/") + "/v1/projects/stackyard/locations/us-central1/shopping_merchant_reviews/sample?stackyard_contract_probe=1"
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
