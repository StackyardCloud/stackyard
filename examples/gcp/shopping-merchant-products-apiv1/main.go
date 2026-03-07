package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	products "cloud.google.com/go/shopping/merchant/products/apiv1"
	"cloud.google.com/go/shopping/merchant/products/apiv1/productspb"
	shoppingtypepb "cloud.google.com/go/shopping/type/typepb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	accountID := getenv("STACKYARD_GCP_MERCHANT_PRODUCTS_ACCOUNT_ID", "123456")
	dataSourceID := getenv("STACKYARD_GCP_MERCHANT_PRODUCTS_DATASOURCE_ID", "104628")
	offerID := getenv("STACKYARD_GCP_MERCHANT_PRODUCTS_OFFER_ID", "sku-1001")
	contentLanguage := strings.ToLower(getenv("STACKYARD_GCP_MERCHANT_PRODUCTS_CONTENT_LANGUAGE", "en"))
	feedLabel := strings.ToUpper(getenv("STACKYARD_GCP_MERCHANT_PRODUCTS_FEED_LABEL", "US"))

	parent := fmt.Sprintf("accounts/%s", accountID)
	dataSource := fmt.Sprintf("%s/dataSources/%s", parent, dataSourceID)
	productID := fmt.Sprintf("%s~%s~%s", contentLanguage, feedLabel, offerID)
	productName := fmt.Sprintf("%s/products/%s", parent, productID)
	productInputName := fmt.Sprintf("%s/productInputs/%s", parent, productID)

	fmt.Printf("Stackyard GCP Shopping Merchant Products shopping/merchant/products/apiv1 clients using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-products-apiv1",
		},
	}

	productsClient, err := products.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant products client: %v", err)
	}
	defer closeClient("merchant products", productsClient.Close)

	productInputsClient, err := products.NewProductInputsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant product inputs client: %v", err)
	}
	defer closeClient("merchant product inputs", productInputsClient.Close)

	insertResp, err := productInputsClient.InsertProductInput(ctx, &productspb.InsertProductInputRequest{
		Parent:     parent,
		DataSource: dataSource,
		ProductInput: &productspb.ProductInput{
			OfferId:         offerID,
			ContentLanguage: contentLanguage,
			FeedLabel:       feedLabel,
			ProductAttributes: &productspb.ProductAttributes{
				Title:       proto.String("Stackyard Product " + offerID),
				Description: proto.String("Inserted product input"),
			},
			CustomAttributes: []*shoppingtypepb.CustomAttribute{
				{
					Name:  proto.String("material"),
					Value: proto.String("cotton"),
				},
			},
		},
	})
	if err != nil {
		exitf("InsertProductInput failed: %v", err)
	}
	if got := strings.TrimSpace(insertResp.GetName()); got == "" {
		exitf("InsertProductInput returned empty name")
	}
	fmt.Printf("InsertProductInput succeeded: %s\n", insertResp.GetName())

	updateResp, err := productInputsClient.UpdateProductInput(ctx, &productspb.UpdateProductInputRequest{
		ProductInput: &productspb.ProductInput{
			Name:            productInputName,
			OfferId:         offerID,
			ContentLanguage: contentLanguage,
			FeedLabel:       feedLabel,
			ProductAttributes: &productspb.ProductAttributes{
				Title: proto.String("Stackyard Product " + offerID + " Updated"),
			},
		},
		DataSource: dataSource,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"product_attributes.title"}},
	})
	if err != nil {
		exitf("UpdateProductInput failed: %v", err)
	}
	if got := updateResp.GetProductAttributes().GetTitle(); strings.TrimSpace(got) == "" {
		exitf("UpdateProductInput returned empty product_attributes.title")
	}
	fmt.Printf("UpdateProductInput succeeded: %s\n", updateResp.GetName())

	getResp, err := productsClient.GetProduct(ctx, &productspb.GetProductRequest{Name: productName})
	if err != nil {
		exitf("GetProduct failed: %v", err)
	}
	if strings.TrimSpace(getResp.GetName()) == "" {
		exitf("GetProduct returned empty name")
	}
	fmt.Printf("GetProduct succeeded: %s\n", getResp.GetName())

	it := productsClient.ListProducts(ctx, &productspb.ListProductsRequest{
		Parent:   parent,
		PageSize: 1,
	})
	first, err := it.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		exitf("ListProducts failed: %v", err)
	}
	if err == nil && first != nil {
		fmt.Printf("ListProducts succeeded: first=%s\n", first.GetName())
	} else {
		fmt.Println("ListProducts succeeded: no products returned")
	}

	if err := productInputsClient.DeleteProductInput(ctx, &productspb.DeleteProductInputRequest{
		Name:       productInputName,
		DataSource: dataSource,
	}); err != nil {
		exitf("DeleteProductInput failed: %v", err)
	}
	fmt.Printf("DeleteProductInput succeeded: %s\n", productInputName)

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, endpoint string) error {
	target := strings.TrimRight(endpoint, "/") + "/v1/projects/stackyard/locations/us-central1/shopping_merchant_products/sample?stackyard_contract_probe=1"
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
