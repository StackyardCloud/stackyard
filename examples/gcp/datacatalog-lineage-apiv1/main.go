package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	lineage "cloud.google.com/go/datacatalog/lineage/apiv1"
	"cloud.google.com/go/datacatalog/lineage/apiv1/lineagepb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
)

type callSpec struct {
	name string
	call func(context.Context, *lineage.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	processID := getenv("STACKYARD_GCP_DATALINEAGE_PROCESS_ID", "team-process")
	runID := getenv("STACKYARD_GCP_DATALINEAGE_RUN_ID", "run-1")
	lineageEventID := getenv("STACKYARD_GCP_DATALINEAGE_EVENT_ID", "event-1")
	linkID := getenv("STACKYARD_GCP_DATALINEAGE_LINK_ID", "link-1")
	operationID := getenv("STACKYARD_GCP_DATALINEAGE_OPERATION_ID", "op-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	processName := locationName + "/processes/" + processID
	runName := processName + "/runs/" + runID
	lineageEventName := runName + "/lineageEvents/" + lineageEventID
	linkName := locationName + "/links/" + linkID
	operationName := locationName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Data Lineage apiv1 client using %s\n", apiEndpoint)

	client, err := lineage.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create lineage client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ProcessOpenLineageRunEvent",
			call: func(ctx context.Context, c *lineage.Client) error {
				_, err := c.ProcessOpenLineageRunEvent(ctx, &lineagepb.ProcessOpenLineageRunEventRequest{
					Parent: locationName,
					OpenLineage: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"eventType": structpb.NewStringValue("COMPLETE"),
						},
					},
				})
				return err
			},
		},
		{
			name: "ListProcesses",
			call: func(ctx context.Context, c *lineage.Client) error {
				it := c.ListProcesses(ctx, &lineagepb.ListProcessesRequest{
					Parent:   locationName,
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
			name: "GetProcess",
			call: func(ctx context.Context, c *lineage.Client) error {
				_, err := c.GetProcess(ctx, &lineagepb.GetProcessRequest{Name: processName})
				return err
			},
		},
		{
			name: "CreateProcess",
			call: func(ctx context.Context, c *lineage.Client) error {
				_, err := c.CreateProcess(ctx, &lineagepb.CreateProcessRequest{
					Parent: locationName,
					Process: &lineagepb.Process{
						Name:        processName,
						DisplayName: "Team Process",
					},
				})
				return err
			},
		},
		{
			name: "UpdateProcess",
			call: func(ctx context.Context, c *lineage.Client) error {
				_, err := c.UpdateProcess(ctx, &lineagepb.UpdateProcessRequest{
					Process: &lineagepb.Process{
						Name:        processName,
						DisplayName: "Team Process Updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "ListRuns",
			call: func(ctx context.Context, c *lineage.Client) error {
				it := c.ListRuns(ctx, &lineagepb.ListRunsRequest{
					Parent:   processName,
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
			name: "GetRun",
			call: func(ctx context.Context, c *lineage.Client) error {
				_, err := c.GetRun(ctx, &lineagepb.GetRunRequest{Name: runName})
				return err
			},
		},
		{
			name: "CreateRun",
			call: func(ctx context.Context, c *lineage.Client) error {
				_, err := c.CreateRun(ctx, &lineagepb.CreateRunRequest{
					Parent: processName,
					Run: &lineagepb.Run{
						Name:        runName,
						DisplayName: "Run 1",
						State:       lineagepb.Run_STARTED,
					},
				})
				return err
			},
		},
		{
			name: "UpdateRun",
			call: func(ctx context.Context, c *lineage.Client) error {
				_, err := c.UpdateRun(ctx, &lineagepb.UpdateRunRequest{
					Run: &lineagepb.Run{
						Name:        runName,
						DisplayName: "Run 1 Updated",
						State:       lineagepb.Run_COMPLETED,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name", "state"}},
				})
				return err
			},
		},
		{
			name: "ListLineageEvents",
			call: func(ctx context.Context, c *lineage.Client) error {
				it := c.ListLineageEvents(ctx, &lineagepb.ListLineageEventsRequest{
					Parent:   runName,
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
			name: "GetLineageEvent",
			call: func(ctx context.Context, c *lineage.Client) error {
				_, err := c.GetLineageEvent(ctx, &lineagepb.GetLineageEventRequest{Name: lineageEventName})
				return err
			},
		},
		{
			name: "CreateLineageEvent",
			call: func(ctx context.Context, c *lineage.Client) error {
				_, err := c.CreateLineageEvent(ctx, &lineagepb.CreateLineageEventRequest{
					Parent: runName,
					LineageEvent: &lineagepb.LineageEvent{
						Name: lineageEventName,
					},
				})
				return err
			},
		},
		{
			name: "SearchLinks",
			call: func(ctx context.Context, c *lineage.Client) error {
				it := c.SearchLinks(ctx, &lineagepb.SearchLinksRequest{
					Parent:   locationName,
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
			name: "BatchSearchLinkProcesses",
			call: func(ctx context.Context, c *lineage.Client) error {
				it := c.BatchSearchLinkProcesses(ctx, &lineagepb.BatchSearchLinkProcessesRequest{
					Parent:   locationName,
					Links:    []string{linkName},
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
			name: "DeleteLineageEvent",
			call: func(ctx context.Context, c *lineage.Client) error {
				return c.DeleteLineageEvent(ctx, &lineagepb.DeleteLineageEventRequest{Name: lineageEventName})
			},
		},
		{
			name: "DeleteRun",
			call: func(ctx context.Context, c *lineage.Client) error {
				_, err := c.DeleteRun(ctx, &lineagepb.DeleteRunRequest{Name: runName})
				return err
			},
		},
		{
			name: "DeleteProcess",
			call: func(ctx context.Context, c *lineage.Client) error {
				_, err := c.DeleteProcess(ctx, &lineagepb.DeleteProcessRequest{Name: processName})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *lineage.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *lineage.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationName,
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
			call: func(ctx context.Context, c *lineage.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *lineage.Client) error {
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
		fmt.Fprintf(os.Stderr, "warning: close lineage client: %v\n", err)
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
