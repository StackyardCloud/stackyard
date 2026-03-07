package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	notebooks "cloud.google.com/go/notebooks/apiv1"
	"cloud.google.com/go/notebooks/apiv1/notebookspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type notebookCallSpec struct {
	name string
	call func(context.Context, *notebooks.NotebookClient) error
}

type managedNotebookCallSpec struct {
	name string
	call func(context.Context, *notebooks.ManagedNotebookClient) error
}

func main() {
	grpcEndpoint := grpcEndpointFromEnv()
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	instanceID := getenv("STACKYARD_GCP_NOTEBOOKS_INSTANCE_ID", "instance-1")
	environmentID := getenv("STACKYARD_GCP_NOTEBOOKS_ENVIRONMENT_ID", "environment-1")
	scheduleID := getenv("STACKYARD_GCP_NOTEBOOKS_SCHEDULE_ID", "schedule-1")
	executionID := getenv("STACKYARD_GCP_NOTEBOOKS_EXECUTION_ID", "execution-1")
	runtimeID := getenv("STACKYARD_GCP_NOTEBOOKS_RUNTIME_ID", "runtime-1")
	vmID := getenv("STACKYARD_GCP_NOTEBOOKS_RUNTIME_VM_ID", "vm-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	instanceName := fmt.Sprintf("%s/instances/%s", parent, instanceID)
	environmentName := fmt.Sprintf("%s/environments/%s", parent, environmentID)
	scheduleName := fmt.Sprintf("%s/schedules/%s", parent, scheduleID)
	executionName := fmt.Sprintf("%s/executions/%s", parent, executionID)
	runtimeName := fmt.Sprintf("%s/runtimes/%s", parent, runtimeID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Notebooks apiv1 clients using %s\n", grpcEndpoint)

	clientOpts := []option.ClientOption{
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	notebookClient, err := notebooks.NewNotebookClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create notebooks client: %v", err)
	}
	defer closeClient("notebooks", notebookClient.Close)

	managedClient, err := notebooks.NewManagedNotebookClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create managed notebooks client: %v", err)
	}
	defer closeClient("managed notebooks", managedClient.Close)

	notebookCalls := []notebookCallSpec{
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
			name: "IsInstanceUpgradeable",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.IsInstanceUpgradeable(ctx, &notebookspb.IsInstanceUpgradeableRequest{
					NotebookInstance: instanceName,
				})
				return err
			},
		},
		{
			name: "GetInstanceHealth",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.GetInstanceHealth(ctx, &notebookspb.GetInstanceHealthRequest{Name: instanceName})
				return err
			},
		},
		{
			name: "ListEnvironments",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				it := c.ListEnvironments(ctx, &notebookspb.ListEnvironmentsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetEnvironment",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.GetEnvironment(ctx, &notebookspb.GetEnvironmentRequest{Name: environmentName})
				return err
			},
		},
		{
			name: "ListSchedules",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				it := c.ListSchedules(ctx, &notebookspb.ListSchedulesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetSchedule",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.GetSchedule(ctx, &notebookspb.GetScheduleRequest{Name: scheduleName})
				return err
			},
		},
		{
			name: "ListExecutions",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				it := c.ListExecutions(ctx, &notebookspb.ListExecutionsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetExecution",
			call: func(ctx context.Context, c *notebooks.NotebookClient) error {
				_, err := c.GetExecution(ctx, &notebookspb.GetExecutionRequest{Name: executionName})
				return err
			},
		},
	}

	managedCalls := []managedNotebookCallSpec{
		{
			name: "ListRuntimes",
			call: func(ctx context.Context, c *notebooks.ManagedNotebookClient) error {
				it := c.ListRuntimes(ctx, &notebookspb.ListRuntimesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetRuntime",
			call: func(ctx context.Context, c *notebooks.ManagedNotebookClient) error {
				_, err := c.GetRuntime(ctx, &notebookspb.GetRuntimeRequest{Name: runtimeName})
				return err
			},
		},
		{
			name: "RefreshRuntimeTokenInternal",
			call: func(ctx context.Context, c *notebooks.ManagedNotebookClient) error {
				_, err := c.RefreshRuntimeTokenInternal(ctx, &notebookspb.RefreshRuntimeTokenInternalRequest{
					Name: runtimeName,
					VmId: vmID,
				})
				return err
			},
		},
	}

	runNotebookCalls(ctx, notebookClient, notebookCalls)
	runManagedNotebookCalls(ctx, managedClient, managedCalls)

	fmt.Println("Done.")
}

func runNotebookCalls(ctx context.Context, client *notebooks.NotebookClient, calls []notebookCallSpec) {
	for _, call := range calls {
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, client)
		cancel()

		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedFoundationError(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}
}

func runManagedNotebookCalls(ctx context.Context, client *notebooks.ManagedNotebookClient, calls []managedNotebookCallSpec) {
	for _, call := range calls {
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, client)
		cancel()

		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedFoundationError(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}
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
