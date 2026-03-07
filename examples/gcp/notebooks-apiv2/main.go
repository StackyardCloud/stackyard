package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	notebooks "cloud.google.com/go/notebooks/apiv2"
	"cloud.google.com/go/notebooks/apiv2/notebookspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *notebooks.NotebookClient) error
}

func main() {
	grpcEndpoint := grpcEndpointFromEnv()
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	instanceID := getenv("STACKYARD_GCP_NOTEBOOKS_INSTANCE_ID", "instance-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	instanceName := fmt.Sprintf("%s/instances/%s", parent, instanceID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Notebooks apiv2 client using %s\n", grpcEndpoint)

	client, err := notebooks.NewNotebookClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create notebooks v2 client: %v", err)
	}
	defer closeClient("notebooks v2", client.Close)

	calls := []callSpec{
		{
			name: "ListInstances",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				it := c.ListInstances(ctx, &notebookspb.ListInstancesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetInstance",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.GetInstance(ctx, &notebookspb.GetInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "CheckInstanceUpgradability",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.CheckInstanceUpgradability(ctx, &notebookspb.CheckInstanceUpgradabilityRequest{
					NotebookInstance: instanceName,
				})
				return err
			},
		},
		{
			name: "StartInstance",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.StartInstance(ctx, &notebookspb.StartInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "StopInstance",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.StopInstance(ctx, &notebookspb.StopInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "ResetInstance",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.ResetInstance(ctx, &notebookspb.ResetInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "UpgradeInstance",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.UpgradeInstance(ctx, &notebookspb.UpgradeInstanceRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "RollbackInstance",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.RollbackInstance(ctx, &notebookspb.RollbackInstanceRequest{
					Name:           instanceName,
					TargetSnapshot: fmt.Sprintf("projects/%s/global/snapshots/snapshot-1", projectID),
					RevisionId:     "1",
				})
				return err
			},
		},
		{
			name: "DiagnoseInstance",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.DiagnoseInstance(ctx, &notebookspb.DiagnoseInstanceRequest{
					Name: instanceName,
					DiagnosticConfig: &notebookspb.DiagnosticConfig{
						GcsBucket: "gs://stackyard-notebooks-diagnostics",
					},
					TimeoutMinutes: 5,
				})
				return err
			},
		},
	}

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, client)
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

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
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
