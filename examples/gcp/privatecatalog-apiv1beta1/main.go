package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	privatecatalog "cloud.google.com/go/privatecatalog/apiv1beta1"
	"cloud.google.com/go/privatecatalog/apiv1beta1/privatecatalogpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *privatecatalog.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	resource := getenv("STACKYARD_GCP_PRIVATECATALOG_RESOURCE", fmt.Sprintf("projects/%s", projectID))
	catalogName := getenv("STACKYARD_GCP_PRIVATECATALOG_CATALOG_NAME", "catalogs/default-catalog")
	productName := getenv("STACKYARD_GCP_PRIVATECATALOG_PRODUCT_NAME", catalogName+"/products/default-product")

	fmt.Printf("Stackyard GCP Cloud Private Catalog apiv1beta1 client using %s\n", apiEndpoint)

	client, err := privatecatalog.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create privatecatalog client: %v", err)
	}
	defer closeClient("privatecatalog", client.Close)

	calls := []callSpec{
		{
			name: "SearchCatalogs",
			call: func(ctx context.Context, c *privatecatalog.Client) error {
				it := c.SearchCatalogs(ctx, &privatecatalogpb.SearchCatalogsRequest{
					Resource: resource,
					Query:    "name=" + catalogName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "SearchProducts",
			call: func(ctx context.Context, c *privatecatalog.Client) error {
				it := c.SearchProducts(ctx, &privatecatalogpb.SearchProductsRequest{
					Resource: resource,
					Query:    "parent=" + catalogName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "SearchVersions",
			call: func(ctx context.Context, c *privatecatalog.Client) error {
				it := c.SearchVersions(ctx, &privatecatalogpb.SearchVersionsRequest{
					Resource: resource,
					Query:    "parent=" + productName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "notimplemented") || strings.Contains(lower, "not implemented")
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
