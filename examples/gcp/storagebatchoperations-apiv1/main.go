package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	storagebatchoperations "cloud.google.com/go/storagebatchoperations/apiv1"
	"cloud.google.com/go/storagebatchoperations/apiv1/storagebatchoperationspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *storagebatchoperations.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	jobID := getenv("STACKYARD_GCP_STORAGE_BATCH_JOB_ID", "job-1")
	bucketOperationID := getenv("STACKYARD_GCP_STORAGE_BATCH_BUCKET_OPERATION_ID", "bucket-op-1")

	parent := fmt.Sprintf("projects/%s/locations/global", projectID)
	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := parent
	jobName := fmt.Sprintf("%s/jobs/%s", parent, jobID)
	bucketOperationName := fmt.Sprintf("%s/bucketOperations/%s", jobName, bucketOperationID)
	operationName := fmt.Sprintf("%s/operations/createJob.%s", parent, jobID)

	fmt.Printf("Stackyard GCP Storage Batch Operations apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "storagebatchoperations",
		},
	}

	client, err := storagebatchoperations.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create storagebatchoperations client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
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
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "CreateJob",
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
				op, err := c.CreateJob(ctx, &storagebatchoperationspb.CreateJobRequest{
					Parent:    parent,
					JobId:     jobID,
					RequestId: "11111111-1111-4111-8111-111111111111",
					Job: &storagebatchoperationspb.Job{
						Name:        jobName,
						Description: "Stackyard storage batch operation job",
						Source: &storagebatchoperationspb.Job_BucketList{
							BucketList: &storagebatchoperationspb.BucketList{
								Buckets: []*storagebatchoperationspb.BucketList_Bucket{
									{Bucket: "stackyard-source-bucket"},
								},
							},
						},
						Transformation: &storagebatchoperationspb.Job_DeleteObject{
							DeleteObject: &storagebatchoperationspb.DeleteObject{
								PermanentObjectDeletionEnabled: true,
							},
						},
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "GetJob",
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
				_, err := c.GetJob(ctx, &storagebatchoperationspb.GetJobRequest{Name: jobName})
				return err
			},
		},
		{
			name: "ListJobs",
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
				it := c.ListJobs(ctx, &storagebatchoperationspb.ListJobsRequest{
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
			name: "CancelJob",
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
				_, err := c.CancelJob(ctx, &storagebatchoperationspb.CancelJobRequest{
					Name:      jobName,
					RequestId: "22222222-2222-4222-8222-222222222222",
				})
				return err
			},
		},
		{
			name: "ListBucketOperations",
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
				it := c.ListBucketOperations(ctx, &storagebatchoperationspb.ListBucketOperationsRequest{
					Parent:   jobName,
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
			name: "GetBucketOperation",
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
				_, err := c.GetBucketOperation(ctx, &storagebatchoperationspb.GetBucketOperationRequest{Name: bucketOperationName})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
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
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteJob",
			call: func(ctx context.Context, c *storagebatchoperations.Client) error {
				return c.DeleteJob(ctx, &storagebatchoperationspb.DeleteJobRequest{
					Name:      jobName,
					RequestId: "33333333-3333-4333-8333-333333333333",
					Force:     true,
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
		fmt.Fprintf(os.Stderr, "warning: close storagebatchoperations client: %v\n", err)
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
