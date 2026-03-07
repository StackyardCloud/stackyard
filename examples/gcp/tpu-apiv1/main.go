package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	tpu "cloud.google.com/go/tpu/apiv1"
	"cloud.google.com/go/tpu/apiv1/tpupb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *tpu.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	apiEndpoint := endpoint + "/gcp"
	grpcEndpoint := strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	nodeID := getenv("STACKYARD_GCP_TPU_NODE_ID", "node-1")
	startableNodeID := getenv("STACKYARD_GCP_TPU_STARTABLE_NODE_ID", "node-stopped")

	projectName := fmt.Sprintf("projects/%s", projectID)
	parent := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	nodeName := fmt.Sprintf("%s/nodes/%s", parent, nodeID)
	startableNodeName := fmt.Sprintf("%s/nodes/%s", parent, startableNodeID)
	tensorFlowVersionName := fmt.Sprintf("%s/tensorflowVersions/v2-alpha", parent)
	acceleratorTypeName := fmt.Sprintf("%s/acceleratorTypes/v3-8", parent)
	operationName := fmt.Sprintf("%s/operations/createNode.%s", parent, nodeID)

	fmt.Printf("Stackyard GCP Cloud TPU V1 tpu/apiv1 client using %s (grpc=%s)\n", apiEndpoint, grpcEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, projectID, locationID); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	client, err := tpu.NewClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create tpu client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *tpu.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
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
			name: "GetLocation",
			call: func(ctx context.Context, c *tpu.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListAcceleratorTypes",
			call: func(ctx context.Context, c *tpu.Client) error {
				it := c.ListAcceleratorTypes(ctx, &tpupb.ListAcceleratorTypesRequest{
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
			name: "GetAcceleratorType",
			call: func(ctx context.Context, c *tpu.Client) error {
				_, err := c.GetAcceleratorType(ctx, &tpupb.GetAcceleratorTypeRequest{Name: acceleratorTypeName})
				return err
			},
		},
		{
			name: "ListTensorFlowVersions",
			call: func(ctx context.Context, c *tpu.Client) error {
				it := c.ListTensorFlowVersions(ctx, &tpupb.ListTensorFlowVersionsRequest{
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
			name: "GetTensorFlowVersion",
			call: func(ctx context.Context, c *tpu.Client) error {
				_, err := c.GetTensorFlowVersion(ctx, &tpupb.GetTensorFlowVersionRequest{Name: tensorFlowVersionName})
				return err
			},
		},
		{
			name: "ListNodes",
			call: func(ctx context.Context, c *tpu.Client) error {
				it := c.ListNodes(ctx, &tpupb.ListNodesRequest{
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
			name: "GetNode",
			call: func(ctx context.Context, c *tpu.Client) error {
				_, err := c.GetNode(ctx, &tpupb.GetNodeRequest{Name: nodeName})
				return err
			},
		},
		{
			name: "CreateNode",
			call: func(ctx context.Context, c *tpu.Client) error {
				op, err := c.CreateNode(ctx, &tpupb.CreateNodeRequest{
					Parent: parent,
					NodeId: nodeID,
					Node: &tpupb.Node{
						Name:              nodeName,
						AcceleratorType:   "v3-8",
						TensorflowVersion: "v2-alpha",
						Description:       "Stackyard TPU node",
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				if _, err := op.Wait(ctx); err != nil {
					return err
				}
				return nil
			},
		},
		{
			name: "StopNode",
			call: func(ctx context.Context, c *tpu.Client) error {
				op, err := c.StopNode(ctx, &tpupb.StopNodeRequest{Name: nodeName})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "StartNode",
			call: func(ctx context.Context, c *tpu.Client) error {
				op, err := c.StartNode(ctx, &tpupb.StartNodeRequest{Name: startableNodeName})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "ReimageNode",
			call: func(ctx context.Context, c *tpu.Client) error {
				op, err := c.ReimageNode(ctx, &tpupb.ReimageNodeRequest{
					Name:              nodeName,
					TensorflowVersion: "v2-alpha",
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "DeleteNode",
			call: func(ctx context.Context, c *tpu.Client) error {
				op, err := c.DeleteNode(ctx, &tpupb.DeleteNodeRequest{Name: nodeName})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *tpu.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *tpu.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent,
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
			name: "CancelOperation",
			call: func(ctx context.Context, c *tpu.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *tpu.Client) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
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

func waitForStackyardReady(ctx context.Context, apiEndpoint, projectID, locationID string) error {
	readyURL := fmt.Sprintf(
		"%s/v1/projects/%s/locations/%s/tpu?stackyard_contract_probe=1&typedSuccess=1",
		strings.TrimRight(apiEndpoint, "/"),
		projectID,
		locationID,
	)
	httpClient := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "tpu")

		resp, err := httpClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("ready probe status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s: %w", readyURL, lastErr)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}
	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return true
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotImplemented {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close tpu client: %v\n", err)
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
