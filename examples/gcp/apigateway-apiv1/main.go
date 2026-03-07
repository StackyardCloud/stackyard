package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	apigateway "cloud.google.com/go/apigateway/apiv1"
	"cloud.google.com/go/apigateway/apiv1/apigatewaypb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *apigateway.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	parentGlobal := getenv("STACKYARD_GCP_PARENT_GLOBAL", "projects/stackyard/locations/global")
	parentGateway := getenv("STACKYARD_GCP_GATEWAY_PARENT", "projects/stackyard/locations/us-central1")
	apiID := getenv("STACKYARD_GCP_API_ID", "team-api")
	configID := getenv("STACKYARD_GCP_CONFIG_ID", "team-config")
	gatewayID := getenv("STACKYARD_GCP_GATEWAY_ID", "team-gateway")

	apiName := parentGlobal + "/apis/" + apiID
	configName := apiName + "/configs/" + configID
	gatewayName := parentGateway + "/gateways/" + gatewayID

	fmt.Printf("Stackyard GCP API Gateway apiv1 client using %s\n", apiEndpoint)

	client, err := apigateway.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create apigateway client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListApis",
			call: func(ctx context.Context, c *apigateway.Client) error {
				it := c.ListApis(ctx, &apigatewaypb.ListApisRequest{
					Parent:   parentGlobal,
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
			name: "GetApi",
			call: func(ctx context.Context, c *apigateway.Client) error {
				_, err := c.GetApi(ctx, &apigatewaypb.GetApiRequest{Name: apiName})
				return err
			},
		},
		{
			name: "ListApiConfigs",
			call: func(ctx context.Context, c *apigateway.Client) error {
				it := c.ListApiConfigs(ctx, &apigatewaypb.ListApiConfigsRequest{
					Parent:   apiName,
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
			name: "GetApiConfig",
			call: func(ctx context.Context, c *apigateway.Client) error {
				_, err := c.GetApiConfig(ctx, &apigatewaypb.GetApiConfigRequest{Name: configName})
				return err
			},
		},
		{
			name: "ListGateways",
			call: func(ctx context.Context, c *apigateway.Client) error {
				it := c.ListGateways(ctx, &apigatewaypb.ListGatewaysRequest{
					Parent:   parentGateway,
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
			name: "GetGateway",
			call: func(ctx context.Context, c *apigateway.Client) error {
				_, err := c.GetGateway(ctx, &apigatewaypb.GetGatewayRequest{Name: gatewayName})
				return err
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

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close apigateway client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
