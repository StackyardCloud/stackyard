package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	executions "cloud.google.com/go/workflows/executions/apiv1"
	executionspb "cloud.google.com/go/workflows/executions/apiv1/executionspb"
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
	call func(context.Context, *executions.Client) error
}

func main() {
	grpcEndpoint := grpcEndpointFromEnv()
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	workflowID := getenv("STACKYARD_GCP_WORKFLOW_ID", "workflow-1")
	executionID := getenv("STACKYARD_GCP_WORKFLOW_EXECUTION_ID", "execution-1")

	parent := fmt.Sprintf("projects/%s/locations/%s/workflows/%s", projectID, locationID, workflowID)
	executionName := parent + "/executions/" + executionID
	createdExecutionName := ""

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Workflow Executions workflows/executions/apiv1 client using %s\n", grpcEndpoint)

	client, err := executions.NewClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create workflow executions client: %v", err)
	}
	defer closeClient("workflow executions", client.Close)

	calls := []callSpec{
		{
			name: "CreateExecution",
			call: func(ctx context.Context, c *executions.Client) error {
				resp, err := c.CreateExecution(ctx, &executionspb.CreateExecutionRequest{
					Parent: parent,
					Execution: &executionspb.Execution{
						Argument:     `{"input":"stackyard"}`,
						CallLogLevel: executionspb.Execution_LOG_ALL_CALLS,
						Labels: map[string]string{
							"env":  "staged",
							"team": "platform",
						},
					},
				})
				if err != nil {
					return err
				}
				createdExecutionName = strings.TrimSpace(resp.GetName())
				return nil
			},
		},
		{
			name: "GetExecution",
			call: func(ctx context.Context, c *executions.Client) error {
				target := firstNonEmpty(createdExecutionName, executionName)
				_, err := c.GetExecution(ctx, &executionspb.GetExecutionRequest{
					Name: target,
					View: executionspb.ExecutionView_FULL,
				})
				return err
			},
		},
		{
			name: "ListExecutions",
			call: func(ctx context.Context, c *executions.Client) error {
				it := c.ListExecutions(ctx, &executionspb.ListExecutionsRequest{
					Parent:   parent,
					PageSize: 1,
					View:     executionspb.ExecutionView_FULL,
					Filter:   `state="ACTIVE"`,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CancelExecution",
			call: func(ctx context.Context, c *executions.Client) error {
				target := firstNonEmpty(createdExecutionName, executionName)
				_, err := c.CancelExecution(ctx, &executionspb.CancelExecutionRequest{Name: target})
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
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
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
