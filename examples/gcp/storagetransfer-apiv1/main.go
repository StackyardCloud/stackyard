package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	storagetransfer "cloud.google.com/go/storagetransfer/apiv1"
	"cloud.google.com/go/storagetransfer/apiv1/storagetransferpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *storagetransfer.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	transferJobID := getenv("STACKYARD_GCP_STORAGE_TRANSFER_JOB_ID", "job-1")
	agentPoolID := getenv("STACKYARD_GCP_STORAGE_TRANSFER_AGENT_POOL_ID", "agentpool-1")

	transferJobName := fmt.Sprintf("transferJobs/%s", transferJobID)
	agentPoolName := fmt.Sprintf("projects/%s/agentPools/%s", projectID, agentPoolID)
	operationName := fmt.Sprintf("transferOperations/run.%s", transferJobID)
	listTransferJobsFilter := fmt.Sprintf(`{"projectId":"%s"}`, projectID)

	fmt.Printf("Stackyard GCP Storage Transfer apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "storagetransfer",
		},
	}

	client, err := storagetransfer.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create storagetransfer client: %v", err)
	}
	defer closeClient(client.Close)

	transferJob := func(description string, state storagetransferpb.TransferJob_Status) *storagetransferpb.TransferJob {
		return &storagetransferpb.TransferJob{
			Name:        transferJobName,
			ProjectId:   projectID,
			Description: description,
			Status:      state,
			TransferSpec: &storagetransferpb.TransferSpec{
				DataSource: &storagetransferpb.TransferSpec_GcsDataSource{
					GcsDataSource: &storagetransferpb.GcsData{
						BucketName: "stackyard-source-bucket",
					},
				},
				DataSink: &storagetransferpb.TransferSpec_GcsDataSink{
					GcsDataSink: &storagetransferpb.GcsData{
						BucketName: "stackyard-destination-bucket",
					},
				},
			},
		}
	}

	agentPool := func(displayName string, limitMbps int64) *storagetransferpb.AgentPool {
		return &storagetransferpb.AgentPool{
			Name:        agentPoolName,
			DisplayName: displayName,
			BandwidthLimit: &storagetransferpb.AgentPool_BandwidthLimit{
				LimitMbps: limitMbps,
			},
		}
	}

	calls := []callSpec{
		{
			name: "GetGoogleServiceAccount",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				_, err := c.GetGoogleServiceAccount(ctx, &storagetransferpb.GetGoogleServiceAccountRequest{
					ProjectId: projectID,
				})
				return err
			},
		},
		{
			name: "CreateTransferJob",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				_, err := c.CreateTransferJob(ctx, &storagetransferpb.CreateTransferJobRequest{
					TransferJob: transferJob("Stackyard transfer job", storagetransferpb.TransferJob_ENABLED),
				})
				return err
			},
		},
		{
			name: "GetTransferJob",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				_, err := c.GetTransferJob(ctx, &storagetransferpb.GetTransferJobRequest{
					JobName:   transferJobName,
					ProjectId: projectID,
				})
				return err
			},
		},
		{
			name: "ListTransferJobs",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				it := c.ListTransferJobs(ctx, &storagetransferpb.ListTransferJobsRequest{
					Filter:   listTransferJobsFilter,
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
			name: "UpdateTransferJob",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				_, err := c.UpdateTransferJob(ctx, &storagetransferpb.UpdateTransferJobRequest{
					JobName:     transferJobName,
					ProjectId:   projectID,
					TransferJob: transferJob("Stackyard updated transfer job", storagetransferpb.TransferJob_DISABLED),
					UpdateTransferJobFieldMask: &fieldmaskpb.FieldMask{
						Paths: []string{"status"},
					},
				})
				return err
			},
		},
		{
			name: "RunTransferJob",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				op, err := c.RunTransferJob(ctx, &storagetransferpb.RunTransferJobRequest{
					JobName:   transferJobName,
					ProjectId: projectID,
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return op.Poll(ctx)
			},
		},
		{
			name: "PauseTransferOperation",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				return c.PauseTransferOperation(ctx, &storagetransferpb.PauseTransferOperationRequest{
					Name: operationName,
				})
			},
		},
		{
			name: "ResumeTransferOperation",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				return c.ResumeTransferOperation(ctx, &storagetransferpb.ResumeTransferOperationRequest{
					Name: operationName,
				})
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     "transferOperations",
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
			name: "GetOperation",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{
					Name: operationName,
				})
				return err
			},
		},
		{
			name: "CreateAgentPool",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				_, err := c.CreateAgentPool(ctx, &storagetransferpb.CreateAgentPoolRequest{
					ProjectId:   projectID,
					AgentPoolId: agentPoolID,
					AgentPool:   agentPool("Stackyard Agent Pool", 250),
				})
				return err
			},
		},
		{
			name: "GetAgentPool",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				_, err := c.GetAgentPool(ctx, &storagetransferpb.GetAgentPoolRequest{
					Name: agentPoolName,
				})
				return err
			},
		},
		{
			name: "ListAgentPools",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				it := c.ListAgentPools(ctx, &storagetransferpb.ListAgentPoolsRequest{
					ProjectId: projectID,
					PageSize:  1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "UpdateAgentPool",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				_, err := c.UpdateAgentPool(ctx, &storagetransferpb.UpdateAgentPoolRequest{
					AgentPool: agentPool("Stackyard Updated Agent Pool", 500),
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"display_name"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteAgentPool",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				return c.DeleteAgentPool(ctx, &storagetransferpb.DeleteAgentPoolRequest{
					Name: agentPoolName,
				})
			},
		},
		{
			name: "DeleteTransferJob",
			call: func(ctx context.Context, c *storagetransfer.Client) error {
				return c.DeleteTransferJob(ctx, &storagetransferpb.DeleteTransferJobRequest{
					JobName:   transferJobName,
					ProjectId: projectID,
				})
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
		fmt.Fprintf(os.Stderr, "warning: close storagetransfer client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
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

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	return base.RoundTrip(clone)
}
