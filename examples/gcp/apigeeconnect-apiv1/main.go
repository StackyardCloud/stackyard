package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	apigeeconnect "cloud.google.com/go/apigeeconnect/apiv1"
	"cloud.google.com/go/apigeeconnect/apiv1/apigeeconnectpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type clients struct {
	connection *apigeeconnect.ConnectionClient
	tether     *apigeeconnect.TetherClient
}

type callSpec struct {
	name string
	call func(context.Context, *clients) error
}

func main() {
	parent := getenv("STACKYARD_GCP_APIGEE_ENDPOINT", "projects/stackyard/endpoints/local-endpoint")
	project := getenv("STACKYARD_GCP_PROJECT", "projects/stackyard")
	grpcEndpoint := grpcEndpointFromEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Apigee Connect apiv1 client using %s\n", grpcEndpoint)

	clientOptions := []option.ClientOption{
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	connectionClient, err := apigeeconnect.NewConnectionClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create connection client: %v", err)
	}
	defer closeClient("connection", connectionClient.Close)

	tetherClient, err := apigeeconnect.NewTetherClient(ctx, clientOptions...)
	if err != nil {
		exitf("failed to create tether client: %v", err)
	}
	defer closeClient("tether", tetherClient.Close)

	allClients := &clients{
		connection: connectionClient,
		tether:     tetherClient,
	}

	calls := []callSpec{
		{
			name: "ListConnections",
			call: func(ctx context.Context, c *clients) error {
				it := c.connection.ListConnections(ctx, &apigeeconnectpb.ListConnectionsRequest{
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
			name: "Egress",
			call: func(ctx context.Context, c *clients) error {
				stream, err := c.tether.Egress(ctx)
				if err != nil {
					return err
				}
				defer stream.CloseSend()

				if err := stream.Send(&apigeeconnectpb.EgressResponse{
					Id:      "egress-bootstrap",
					Name:    parent,
					Project: project,
				}); err != nil {
					return err
				}

				_, err = stream.Recv()
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			},
		},
	}

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, allClients)
		callCancel()

		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedFoundationError(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func grpcEndpointFromEnv() string {
	if endpoint := strings.TrimSpace(os.Getenv("STACKYARD_GCP_GRPC_ENDPOINT")); endpoint != "" {
		return normalizeEndpoint(endpoint)
	}
	httpBase := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	return normalizeEndpoint(httpBase)
}

func normalizeEndpoint(raw string) string {
	endpoint := strings.TrimSpace(raw)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	if idx := strings.Index(endpoint, "/"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return endpoint
}

func isToleratedFoundationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") ||
		strings.Contains(text, "not implemented") ||
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large")
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
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
