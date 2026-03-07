package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	apikeys "cloud.google.com/go/apikeys/apiv2"
	"cloud.google.com/go/apikeys/apiv2/apikeyspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *apikeys.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	parent := getenv("STACKYARD_GCP_PARENT", "projects/stackyard/locations/global")
	keyID := getenv("STACKYARD_GCP_KEY_ID", "team-key")
	keyName := parent + "/keys/" + keyID
	keyString := getenv("STACKYARD_GCP_KEY_STRING", "stackyard-demo-key")

	fmt.Printf("Stackyard GCP API Keys apiv2 client using %s\n", apiEndpoint)

	client, err := apikeys.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create apikeys client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListKeys",
			call: func(ctx context.Context, c *apikeys.Client) error {
				it := c.ListKeys(ctx, &apikeyspb.ListKeysRequest{
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
			name: "GetKey",
			call: func(ctx context.Context, c *apikeys.Client) error {
				_, err := c.GetKey(ctx, &apikeyspb.GetKeyRequest{Name: keyName})
				return err
			},
		},
		{
			name: "GetKeyString",
			call: func(ctx context.Context, c *apikeys.Client) error {
				_, err := c.GetKeyString(ctx, &apikeyspb.GetKeyStringRequest{Name: keyName})
				return err
			},
		},
		{
			name: "LookupKey",
			call: func(ctx context.Context, c *apikeys.Client) error {
				_, err := c.LookupKey(ctx, &apikeyspb.LookupKeyRequest{KeyString: keyString})
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
		fmt.Fprintf(os.Stderr, "warning: close apikeys client: %v\n", err)
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
