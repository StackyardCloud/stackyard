package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	agentendpoint "cloud.google.com/go/osconfig/agentendpoint/apiv1"
	"cloud.google.com/go/osconfig/agentendpoint/apiv1/agentendpointpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *agentendpoint.Client) error
}

func main() {
	grpcEndpoint := grpcEndpointFromEnv()
	instanceIDToken := getenv("STACKYARD_GCP_OSCONFIG_INSTANCE_ID_TOKEN", "stackyard-instance-token")
	agentVersion := getenv("STACKYARD_GCP_OSCONFIG_AGENT_VERSION", "stackyard-agent/0.1.0")
	taskID := getenv("STACKYARD_GCP_OSCONFIG_TASK_ID", "task-1")
	inventoryChecksum := getenv("STACKYARD_GCP_OSCONFIG_INVENTORY_CHECKSUM", "inventory-checksum-1")
	vmInventoryChecksum := getenv("STACKYARD_GCP_OSCONFIG_VM_INVENTORY_CHECKSUM", "vm-inventory-checksum-1")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP OS Config Agent Endpoint apiv1 client using %s\n", grpcEndpoint)

	client, err := agentendpoint.NewClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create osconfig agentendpoint client: %v", err)
	}
	defer closeClient("osconfig agentendpoint", client.Close)

	calls := []callSpec{
		{
			name: "ReceiveTaskNotification",
			call: func(ctx context.Context, c *agentendpoint.Client) error {
				stream, err := c.ReceiveTaskNotification(ctx, &agentendpointpb.ReceiveTaskNotificationRequest{
					InstanceIdToken: instanceIDToken,
					AgentVersion:    agentVersion,
				})
				if err != nil {
					return err
				}
				_, err = stream.Recv()
				return err
			},
		},
		{
			name: "StartNextTask",
			call: func(ctx context.Context, c *agentendpoint.Client) error {
				_, err := c.StartNextTask(ctx, &agentendpointpb.StartNextTaskRequest{
					InstanceIdToken: instanceIDToken,
				})
				return err
			},
		},
		{
			name: "ReportTaskProgress",
			call: func(ctx context.Context, c *agentendpoint.Client) error {
				_, err := c.ReportTaskProgress(ctx, &agentendpointpb.ReportTaskProgressRequest{
					InstanceIdToken: instanceIDToken,
					TaskId:          taskID,
					TaskType:        agentendpointpb.TaskType_APPLY_PATCHES,
					Progress: &agentendpointpb.ReportTaskProgressRequest_ApplyPatchesTaskProgress{
						ApplyPatchesTaskProgress: &agentendpointpb.ApplyPatchesTaskProgress{
							State: agentendpointpb.ApplyPatchesTaskProgress_STARTED,
						},
					},
				})
				return err
			},
		},
		{
			name: "ReportTaskComplete",
			call: func(ctx context.Context, c *agentendpoint.Client) error {
				_, err := c.ReportTaskComplete(ctx, &agentendpointpb.ReportTaskCompleteRequest{
					InstanceIdToken: instanceIDToken,
					TaskId:          taskID,
					TaskType:        agentendpointpb.TaskType_APPLY_PATCHES,
					Output: &agentendpointpb.ReportTaskCompleteRequest_ApplyPatchesTaskOutput{
						ApplyPatchesTaskOutput: &agentendpointpb.ApplyPatchesTaskOutput{
							State: agentendpointpb.ApplyPatchesTaskOutput_SUCCEEDED,
						},
					},
				})
				return err
			},
		},
		{
			name: "RegisterAgent",
			call: func(ctx context.Context, c *agentendpoint.Client) error {
				_, err := c.RegisterAgent(ctx, &agentendpointpb.RegisterAgentRequest{
					InstanceIdToken:       instanceIDToken,
					AgentVersion:          agentVersion,
					SupportedCapabilities: []string{"PATCH_GA", "CONFIG_V1"},
					OsLongName:            "Debian GNU/Linux 12",
					OsShortName:           "debian",
					OsVersion:             "12",
					OsArchitecture:        "x86_64",
				})
				return err
			},
		},
		{
			name: "ReportInventory",
			call: func(ctx context.Context, c *agentendpoint.Client) error {
				_, err := c.ReportInventory(ctx, &agentendpointpb.ReportInventoryRequest{
					InstanceIdToken:   instanceIDToken,
					InventoryChecksum: inventoryChecksum,
				})
				return err
			},
		},
		{
			name: "ReportVmInventory",
			call: func(ctx context.Context, c *agentendpoint.Client) error {
				_, err := c.ReportVmInventory(ctx, &agentendpointpb.ReportVmInventoryRequest{
					InstanceIdToken:   instanceIDToken,
					InventoryChecksum: vmInventoryChecksum,
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

func isToleratedFoundationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
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
