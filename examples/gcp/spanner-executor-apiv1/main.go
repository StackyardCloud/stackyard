package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	executor "cloud.google.com/go/spanner/executor/apiv1"
	"cloud.google.com/go/spanner/executor/apiv1/executorpb"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	apiEndpoint := endpoint + "/gcp"
	grpcEndpoint := strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	location := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	instanceID := getenv("STACKYARD_GCP_SPANNER_INSTANCE", "stackyard-instance")
	databaseID := getenv("STACKYARD_GCP_SPANNER_DATABASE", "stackyard-db")

	databasePath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, databaseID)
	changeStreamName := databasePath + "/changeStreams/users"

	fmt.Printf("Stackyard GCP Spanner Executor apiv1 client using %s (grpc=%s)\n", apiEndpoint, grpcEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, projectID, location); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	client, err := executor.NewSpannerExecutorProxyClient(
		ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create spanner executor client: %v", err)
	}
	defer closeClient(client.Close)

	stream, err := client.ExecuteActionAsync(ctx)
	if err != nil {
		exitf("ExecuteActionAsync failed: %v", err)
	}

	requests := []*executorpb.SpannerAsyncActionRequest{
		{
			ActionId: 1,
			Action: &executorpb.SpannerAction{
				DatabasePath: databasePath,
				Action: &executorpb.SpannerAction_Start{
					Start: &executorpb.StartTransactionAction{},
				},
			},
		},
		{
			ActionId: 2,
			Action: &executorpb.SpannerAction{
				DatabasePath: databasePath,
				Action: &executorpb.SpannerAction_Read{
					Read: &executorpb.ReadAction{
						Table:  "Users",
						Column: []string{"id", "name"},
						Keys:   &executorpb.KeySet{All: true},
					},
				},
			},
		},
		{
			ActionId: 3,
			Action: &executorpb.SpannerAction{
				DatabasePath: databasePath,
				Action: &executorpb.SpannerAction_Query{
					Query: &executorpb.QueryAction{Sql: "SELECT 1"},
				},
			},
		},
		{
			ActionId: 4,
			Action: &executorpb.SpannerAction{
				DatabasePath: databasePath,
				Action: &executorpb.SpannerAction_Admin{
					Admin: &executorpb.AdminAction{
						Action: &executorpb.AdminAction_ListCloudInstances{
							ListCloudInstances: &executorpb.ListCloudInstancesAction{},
						},
					},
				},
			},
		},
		{
			ActionId: 5,
			Action: &executorpb.SpannerAction{
				DatabasePath: databasePath,
				Action: &executorpb.SpannerAction_GenerateDbPartitionsQuery{
					GenerateDbPartitionsQuery: &executorpb.GenerateDbPartitionsForQueryAction{
						Query: &executorpb.QueryAction{Sql: "SELECT * FROM Users"},
					},
				},
			},
		},
		{
			ActionId: 6,
			Action: &executorpb.SpannerAction{
				DatabasePath: databasePath,
				Action: &executorpb.SpannerAction_ExecutePartition{
					ExecutePartition: &executorpb.ExecutePartitionAction{
						Partition: &executorpb.BatchPartition{
							PartitionToken: []byte("token-1"),
							Table:          strPtr("Users"),
						},
					},
				},
			},
		},
		{
			ActionId: 7,
			Action: &executorpb.SpannerAction{
				DatabasePath: databasePath,
				Action: &executorpb.SpannerAction_ExecuteChangeStreamQuery{
					ExecuteChangeStreamQuery: &executorpb.ExecuteChangeStreamQuery{
						Name:      changeStreamName,
						StartTime: timestamppb.Now(),
					},
				},
			},
		},
		{
			ActionId: 8,
			Action: &executorpb.SpannerAction{
				DatabasePath: databasePath,
				Action: &executorpb.SpannerAction_QueryCancellation{
					QueryCancellation: &executorpb.QueryCancellationAction{
						LongRunningSql: "SELECT * FROM Users",
						CancelQuery:    "CANCEL QUERY ?",
					},
				},
			},
		},
	}

	for _, req := range requests {
		if err := stream.Send(req); err != nil {
			exitf("send action %d failed: %v", req.GetActionId(), err)
		}
		logf("Send action %d succeeded", req.GetActionId())
	}
	if err := stream.CloseSend(); err != nil {
		exitf("close send failed: %v", err)
	}

	received := map[int32]*executorpb.SpannerAsyncActionResponse{}
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			exitf("recv failed: %v", err)
		}
		received[resp.GetActionId()] = resp
		logf("Recv action %d succeeded", resp.GetActionId())
	}

	for _, req := range requests {
		resp, ok := received[req.GetActionId()]
		if !ok {
			exitf("missing response for action %d", req.GetActionId())
		}
		if resp.GetOutcome() == nil || resp.GetOutcome().GetStatus() == nil || resp.GetOutcome().GetStatus().GetCode() != 0 {
			exitf("action %d returned non-success outcome: %#v", req.GetActionId(), resp.GetOutcome())
		}
	}

	if received[2].GetOutcome().GetReadResult() == nil || strings.TrimSpace(received[2].GetOutcome().GetReadResult().GetTable()) == "" {
		exitf("read action returned empty read result")
	}
	if received[4].GetOutcome().GetAdminResult() == nil || received[4].GetOutcome().GetAdminResult().GetInstanceResponse() == nil {
		exitf("admin action returned empty admin result")
	}
	if len(received[5].GetOutcome().GetDbPartition()) == 0 {
		exitf("partition generation action returned no partitions")
	}
	if len(received[7].GetOutcome().GetChangeStreamRecords()) == 0 {
		exitf("change stream action returned no records")
	}

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, apiEndpoint, projectID, location string) error {
	readyURL := fmt.Sprintf("%s/v1/projects/%s/locations/%s/spanner_executor?stackyard_contract_probe=1&typedSuccess=1", strings.TrimRight(apiEndpoint, "/"), projectID, location)
	httpClient := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "spanner-executor")

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

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close spanner executor client: %v\n", err)
	}
}

func strPtr(v string) *string {
	return &v
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
